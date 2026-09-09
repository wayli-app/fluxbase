# syntax=docker/dockerfile:1
FROM golang:1.26.1-bookworm

# Avoid warnings by switching to noninteractive
ENV DEBIAN_FRONTEND=noninteractive

# =============================================================================
# LAYER 1: Base system packages (rarely changes)
# =============================================================================
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl wget vim nano htop build-essential pkg-config \
    postgresql-client redis-tools httpie jq tree zip unzip sudo \
    locales ca-certificates gnupg lsb-release software-properties-common \
    zsh lsof \
    # OCR dependencies for knowledge base PDF extraction
    tesseract-ocr libtesseract-dev libleptonica-dev tesseract-ocr-eng poppler-utils \
    # Image processing library for govips
    libvips-dev \
    # Playwright Chromium headless browser dependencies
    libnss3 libnspr4 libatk1.0-0 libatk-bridge2.0-0 libcups2 \
    libxcomposite1 libxdamage1 libxrandr2 libgbm1 libpango-1.0-0 \
    libcairo2 libasound2 libxshmfence1 \
    # Graphviz for rendering dependency graphs
    graphviz \
    # Hex dump utility
    xxd \
    # Browser for MCP-based debugging
    firefox-esr \
    && rm -rf /var/lib/apt/lists/* \
    && echo "en_US.UTF-8 UTF-8" > /etc/locale.gen \
    && locale-gen en_US.UTF-8 \
    && update-locale LANG=en_US.UTF-8

ENV LANG=en_US.UTF-8 LANGUAGE=en_US:en LC_ALL=en_US.UTF-8

# =============================================================================
# LAYER 2: Docker CLI (rarely changes)
# =============================================================================
RUN install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get install -y --no-install-recommends docker-ce-cli docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

# =============================================================================
# LAYER 3: Node.js runtime (for Astro docs and other Node.js tools)
# =============================================================================
ARG NODE_VERSION=22
RUN curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/* \
    && node --version \
    && npm --version

# =============================================================================
# LAYER 4: Bun runtime (includes Node.js compatibility, rarely changes)
# =============================================================================
ARG BUN_VERSION=1.3.11
ENV BUN_INSTALL=/home/vscode/.bun
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then BUN_ARCH="x64"; \
    elif [ "$ARCH" = "aarch64" ]; then BUN_ARCH="aarch64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi \
    && curl -fsSL "https://github.com/oven-sh/bun/releases/download/bun-v${BUN_VERSION}/bun-linux-${BUN_ARCH}.zip" -o /tmp/bun.zip \
    && unzip -q /tmp/bun.zip -d /tmp \
    && mkdir -p /usr/local/bun/bin \
    && mv /tmp/bun-linux-${BUN_ARCH}/bun /usr/local/bun/bin/ \
    && ln -s /usr/local/bun/bin/bun /usr/local/bin/bun \
    && ln -s /usr/local/bun/bin/bun /usr/local/bin/bunx \
    && rm -rf /tmp/bun.zip /tmp/bun-linux-${BUN_ARCH} \
    && bun --version

# =============================================================================
# LAYER 5: Pinned runtime versions (changes when you update versions)
# =============================================================================
# Pin Deno version instead of latest
ARG DENO_VERSION=2.7.4
RUN ARCH=$(dpkg --print-architecture) \
    && if [ "$ARCH" = "amd64" ]; then DENO_ARCH="x86_64-unknown-linux-gnu"; \
    elif [ "$ARCH" = "arm64" ]; then DENO_ARCH="aarch64-unknown-linux-gnu"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi \
    && curl -fsSL "https://github.com/denoland/deno/releases/download/v${DENO_VERSION}/deno-${DENO_ARCH}.zip" -o /tmp/deno.zip \
    && unzip -q /tmp/deno.zip -d /usr/local/bin \
    && chmod +x /usr/local/bin/deno \
    && rm /tmp/deno.zip \
    && deno --version

# =============================================================================
# LAYER 6: pgschema - Declarative schema migration tool
# =============================================================================
ARG PGSCHEMA_VERSION=1.12.4
RUN ARCH=$(dpkg --print-architecture) \
    && if [ "$ARCH" = "amd64" ]; then PGSCHEMA_ARCH="linux-amd64"; \
    elif [ "$ARCH" = "arm64" ]; then PGSCHEMA_ARCH="linux-arm64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi \
    && curl -fsSL "https://github.com/pgplex/pgschema/releases/download/v${PGSCHEMA_VERSION}/pgschema-${PGSCHEMA_VERSION}-${PGSCHEMA_ARCH}" -o /usr/local/bin/pgschema \
    && chmod +x /usr/local/bin/pgschema \
    && pgschema --help > /dev/null 2>&1 || echo "pgschema installed"

# =============================================================================
# LAYER 7: Create user (rarely changes)
# =============================================================================
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME -s /bin/zsh \
    && echo "$USERNAME ALL=(root) NOPASSWD:ALL" > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

# =============================================================================
# LAYER 7: User setup - global tools (moderately stable)
# =============================================================================
USER $USERNAME
ENV SHELL=/bin/bash \
    BUN_INSTALL=/home/vscode/.bun

# Install global tools with bun (pin versions for cache stability)
RUN bun install -g \
    typescript@5 \
    @types/node@22 \
    eslint@9 \
    prettier@3 \
    @swc/cli@0 \
    @swc/core@1 \
    tsx@4 \
    nodemon@3 \
    concurrently@9 \
    npm-check-updates@17 \
    serve@14

# =============================================================================
# LAYER 8: Go tools (pin versions for cache stability)
# =============================================================================
USER root

# Go tools with pinned versions
RUN go install -v golang.org/x/tools/gopls@latest \
    && go install -v github.com/go-delve/delve/cmd/dlv@latest \
    && go install -v github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest \
    && go install -v github.com/air-verse/air@latest \
    && go install -v -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
    && go install -v github.com/swaggo/swag/cmd/swag@latest \
    && go install -v github.com/vektra/mockery/v2@latest \
    && go install -v github.com/onsi/ginkgo/v2/ginkgo@latest \
    && go install -v gotest.tools/gotestsum@latest \
    && go install -v github.com/rakyll/hey@latest \
    && go install -v github.com/securego/gosec/v2/cmd/gosec@latest \
    && go install -v github.com/mgechev/revive@latest \
    && go install -v github.com/kisielk/errcheck@latest \
    && go install -v honnef.co/go/tools/cmd/staticcheck@latest \
    && go install -v mvdan.cc/gofumpt@latest \
    && go install -v github.com/evilmartians/lefthook@latest \
    && go install -v github.com/ofabry/go-callvis@latest \
    && go install -v github.com/kisielk/godepgraph@latest \
    && go install -v github.com/loov/goda@latest \
    && go install -v github.com/jfeliu007/goplantuml/cmd/goplantuml@latest \
    && go install -v github.com/jorgerojas26/lazysql@latest

# =============================================================================
# LAYER 9: Final setup (frequently changes - Air config, etc.)
# =============================================================================
RUN mkdir -p /go/pkg /go/bin \
    && chown -R $USERNAME:$USERNAME /go

WORKDIR /workspace

ENV GOPATH=/go \
    CGO_ENABLED=1 \
    GO111MODULE=on \
    PATH="/home/vscode/.local/bin:${BUN_INSTALL}/bin:/go/bin:${PATH}"

USER $USERNAME

# Oh My Zsh (after user setup, before final config)
RUN sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended \
    && echo 'export PATH="/go/bin:$HOME/go/bin:$PATH"' >> ~/.zshrc \
    && echo 'export GOPATH="/go"' >> ~/.zshrc

# Air config (this can change, so it's near the end)
RUN mkdir -p /home/$USERNAME/.air
COPY --chown=$USERNAME:$USERNAME <<EOF /home/$USERNAME/.air/.air.toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
args_bin = []
bin = "./tmp/main"
cmd = "go build -o ./tmp/main cmd/fluxbase/main.go"
delay = 1000
exclude_dir = ["assets", "tmp", "vendor", "testdata", "docs", "scripts", ".git", ".github", "dist", "node_modules"]
exclude_file = []
exclude_regex = ["_test.go"]
exclude_unchanged = false
follow_symlink = false
full_bin = ""
include_dir = []
include_ext = ["go", "tpl", "tmpl", "html", "yaml", "yml", "toml", "env"]
include_file = []
kill_delay = "0s"
log = "build-errors.log"
poll = false
poll_interval = 0
rerun = false
rerun_delay = 500
send_interrupt = false
stop_on_error = false

[color]
app = ""
build = "yellow"
main = "magenta"
runner = "green"
watcher = "cyan"

[log]
main_only = false
time = false

[misc]
clean_on_exit = false

[screen]
clear_on_rebuild = false
keep_scroll = true
EOF

ENV DEBIAN_FRONTEND=dialog
