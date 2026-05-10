package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/mem"
)

type denoArgsConfig struct {
	RuntimeType   RuntimeType
	DenoPath      string
	PublicURL     string
	MemoryLimitMB int
	UserToken     string
	ServiceToken  string
}

type outputResult struct {
	stdout       strings.Builder
	stderr       strings.Builder
	truncated    bool
	resultLine   string
	totalSize    int
	maxOutput    int
}

func buildDenoArgs(cfg denoArgsConfig, permissions Permissions, secrets map[string]string, tmpPath string) ([]string, int, uint64) {
	args := []string{"run"}

	memoryLimitMB := permissions.MemoryLimitMB
	if memoryLimitMB <= 0 {
		memoryLimitMB = cfg.MemoryLimitMB
	}

	var availableMemoryMB uint64
	if memoryLimitMB > 0 {
		if vmStat, err := mem.VirtualMemory(); err == nil {
			availableMemoryMB = vmStat.Available / 1024 / 1024
			totalMemoryMB := vmStat.Total / 1024 / 1024

			if uint64(memoryLimitMB) > availableMemoryMB {
				log.Warn().
					Int("requested_memory_mb", memoryLimitMB).
					Uint64("available_memory_mb", availableMemoryMB).
					Uint64("total_memory_mb", totalMemoryMB).
					Msg("Memory limit exceeds available system memory - OOM kill is likely")
			}
		}

		args = append(args, fmt.Sprintf("--v8-flags=--max-old-space-size=%d", memoryLimitMB))
	}

	if permissions.AllowNet || (cfg.UserToken != "" || cfg.ServiceToken != "") {
		allowedDomains := buildNetworkAllowList(permissions, cfg.PublicURL)
		if len(allowedDomains) > 0 {
			args = append(args, fmt.Sprintf("--allow-net=%s", strings.Join(allowedDomains, ",")))
		} else {
			args = append(args, "--allow-net")
		}
	}
	if permissions.AllowEnv {
		args = append(args, "--allow-env")
	} else {
		secretNames := make([]string, 0, len(secrets))
		for name := range secrets {
			secretNames = append(secretNames, name)
		}
		args = append(args, fmt.Sprintf("--allow-env=%s", allowedEnvVars(cfg.RuntimeType, secretNames)))
	}
	if permissions.AllowRead {
		args = append(args, "--allow-read=/tmp")
	}
	if permissions.AllowWrite {
		args = append(args, "--allow-write=/tmp")
	}

	args = append(args, tmpPath)

	return args, memoryLimitMB, availableMemoryMB
}

func startAndStreamOutput(cmd *exec.Cmd, id uuid.UUID, maxOutputSize int, onProgress func(uuid.UUID, *Progress), onLog func(uuid.UUID, string, string)) (*outputResult, error) {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		_ = stdoutPipe.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdoutPipe.Close()
		_ = stderrPipe.Close()
		return nil, fmt.Errorf("failed to start deno: %w", err)
	}

	out := &outputResult{maxOutput: maxOutputSize}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().Interface("panic", rec).Str("id", id.String()).Msg("Panic in stdout processing - recovered")
			}
			wg.Done()
		}()
		scanner := bufio.NewScanner(stdoutPipe)
		const maxLineSize = 1024 * 1024
		scanner.Buffer(make([]byte, maxLineSize), maxLineSize)

		for scanner.Scan() {
			line := scanner.Text()
			lineLen := len(line) + 1

			if strings.HasPrefix(line, "__RESULT__::") {
				out.resultLine = line
			}

			if out.maxOutput > 0 && out.totalSize+lineLen > out.maxOutput {
				if !out.truncated {
					out.truncated = true
					fmt.Fprintf(&out.stdout, "\n[OUTPUT TRUNCATED: exceeded %d bytes limit]\n", out.maxOutput)
					log.Warn().
						Str("id", id.String()).
						Int("max_output_size", out.maxOutput).
						Int("total_output_size", out.totalSize).
						Msg("Function output truncated - exceeded size limit")
				}
			} else if !out.truncated {
				out.stdout.WriteString(line + "\n")
				out.totalSize += lineLen
			}

			if strings.HasPrefix(line, "__PROGRESS__::") {
				progressJSON := strings.TrimPrefix(line, "__PROGRESS__::")
				var progress Progress
				if err := json.Unmarshal([]byte(progressJSON), &progress); err == nil {
					if onProgress != nil {
						onProgress(id, &progress)
					}
				}
			} else if line != "" {
				if onLog != nil {
					onLog(id, "info", line)
				}
			}
		}

		if out.truncated && out.resultLine != "" {
			out.stdout.WriteString(out.resultLine + "\n")
		}

		if err := scanner.Err(); err != nil {
			log.Warn().
				Err(err).
				Str("id", id.String()).
				Msg("Scanner error while reading stdout - result line may be truncated")
		}
	}()

	wg.Add(1)
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().Interface("panic", rec).Str("id", id.String()).Msg("Panic in stderr processing - recovered")
			}
			wg.Done()
		}()
		scanner := bufio.NewScanner(stderrPipe)
		const maxLineSize = 1024 * 1024
		scanner.Buffer(make([]byte, maxLineSize), maxLineSize)

		for scanner.Scan() {
			line := scanner.Text()
			out.stderr.WriteString(line + "\n")

			if onLog != nil && line != "" {
				level := classifyStderrLine(line)
				onLog(id, level, line)
			}
		}

		if err := scanner.Err(); err != nil {
			log.Warn().
				Err(err).
				Str("id", id.String()).
				Msg("Scanner error while reading stderr")
		}
	}()

	cmdErr := cmd.Wait()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Warn().
			Str("id", id.String()).
			Msg("Timeout waiting for output scanners - continuing")
	}

	return out, cmdErr
}
