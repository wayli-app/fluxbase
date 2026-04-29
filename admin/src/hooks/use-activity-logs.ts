/**
 * Hook for fetching and subscribing to real-time activity logs
 * Displays system-wide activity in the admin dashboard
 */
import { useEffect, useState, useRef, useCallback } from "react";
import { logsApi } from "@/lib/api";
import { fluxbaseClient } from "@/lib/fluxbase-client";

// Allowed categories for activity feed (audit trail focus)
const ACTIVITY_CATEGORIES = new Set(["security", "execution", "system"]);

// Allowed log levels (exclude debug/trace noise)
const ACTIVITY_LEVELS = new Set(["warn", "error", "fatal", "panic"]);

// Patterns to exclude from activity feed (websocket noise)
const EXCLUDED_MESSAGE_PATTERNS = [
  /subscription/i,
  /websocket/i,
  /connection (established|closed)/i,
];

/**
 * Check if a log entry should be shown in the activity feed
 */
function isActivityLog(log: ActivityLog): boolean {
  // Check category and level
  if (
    !ACTIVITY_CATEGORIES.has(log.category) ||
    !ACTIVITY_LEVELS.has(log.level)
  ) {
    return false;
  }

  // Filter out websocket/subscription noise
  if (EXCLUDED_MESSAGE_PATTERNS.some((pattern) => pattern.test(log.message))) {
    return false;
  }

  return true;
}

export interface ActivityLog {
  id: string;
  timestamp: string;
  category: string;
  level: string;
  message: string;
  component?: string;
  custom_category?: string;
  request_id?: string;
  trace_id?: string;
  user_id?: string;
  ip_address?: string;
  fields?: Record<string, unknown>;
}

interface UseActivityLogsOptions {
  /** Only subscribe when enabled (e.g., when page is visible) */
  enabled?: boolean;
  /** Maximum number of logs to keep in memory */
  maxLogs?: number;
  /** Time range for initial query (default: 24 hours) */
  timeRangeHours?: number;
  /** Callback when a new log arrives */
  onNewLog?: (log: ActivityLog) => void;
  /** Callback when WebSocket subscription is established */
  onSubscribed?: () => void;
}

interface UseActivityLogsResult {
  /** Current list of activity logs */
  logs: ActivityLog[];
  /** Whether initial logs are being loaded */
  loading: boolean;
  /** Any error that occurred */
  error: Error | null;
  /** Whether WebSocket subscription is active */
  isSubscribed: boolean;
  /** Manually refetch logs */
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching and subscribing to activity logs in real-time.
 *
 * @example
 * ```tsx
 * const { logs, loading, isSubscribed } = useActivityLogs({
 *   enabled: true,
 *   maxLogs: 50,
 *   onNewLog: (log) => console.log('New log:', log.message),
 * })
 * ```
 */
export function useActivityLogs({
  enabled = true,
  maxLogs = 50,
  timeRangeHours = 24,
  onNewLog,
  onSubscribed,
}: UseActivityLogsOptions = {}): UseActivityLogsResult {
  const [logs, setLogs] = useState<ActivityLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [isSubscribed, setIsSubscribed] = useState(false);

  const channelRef = useRef<ReturnType<
    typeof fluxbaseClient.realtime.channel
  > | null>(null);
  const onNewLogRef = useRef(onNewLog);
  const onSubscribedRef = useRef(onSubscribed);

  // Keep callback refs up to date
  useEffect(() => {
    onNewLogRef.current = onNewLog;
  }, [onNewLog]);

  useEffect(() => {
    onSubscribedRef.current = onSubscribed;
  }, [onSubscribed]);

  // Fetch initial logs from the API
  const fetchLogs = useCallback(async () => {
    if (!enabled) return;

    setLoading(true);
    setError(null);

    try {
      const startTime = new Date(
        Date.now() - timeRangeHours * 60 * 60 * 1000,
      ).toISOString();

      const data = await logsApi.query({
        start_time: startTime,
        limit: maxLogs * 2, // Fetch more to account for filtering
        levels: ["warn", "error", "fatal", "panic"],
        hide_static_assets: true,
      });

      const filtered = (data.entries ?? []).filter(isActivityLog);
      setLogs(filtered);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  }, [enabled, maxLogs, timeRangeHours]);

  // Subscribe to real-time log updates
  useEffect(() => {
    if (!enabled) {
      setIsSubscribed(false);
      return;
    }

    // Fetch initial logs
    fetchLogs();

    try {
      // Subscribe to all-logs channel for admin dashboard
      const channel = fluxbaseClient.realtime
        .channel("fluxbase:all_logs")
        .on("broadcast", { event: "log_entry" }, (payload) => {
          // Payload structure: { event: 'log_entry', payload: LogStreamEvent }
          const event = payload.payload as ActivityLog;

          // Filter to only show relevant activity categories
          if (!isActivityLog(event)) {
            return;
          }

          setLogs((prev) => {
            // Avoid duplicates by ID
            if (prev.some((l) => l.id === event.id)) {
              return prev;
            }
            // Add new log and keep only maxLogs most recent
            const updated = [...prev, event];
            if (updated.length > maxLogs) {
              return updated.slice(-maxLogs);
            }
            return updated;
          });

          // Notify callback
          onNewLogRef.current?.(event);
        })
        .subscribe((status) => {
          if (status === "SUBSCRIBED") {
            setIsSubscribed(true);
            setError(null);
            onSubscribedRef.current?.();
          } else if (status === "CHANNEL_ERROR" || status === "TIMED_OUT") {
            setError(new Error(`Subscription ${status}`));
            setIsSubscribed(false);
          }
        });

      channelRef.current = channel;

      return () => {
        channel.unsubscribe();
        channelRef.current = null;
        setIsSubscribed(false);
      };
    } catch (err) {
      setError(err as Error);
      setIsSubscribed(false);
      return undefined;
    }
  }, [enabled, fetchLogs, maxLogs]);

  // Clear logs when disabled
  useEffect(() => {
    if (!enabled) {
      setLogs([]);
      setError(null);
    }
  }, [enabled]);

  return {
    logs,
    loading,
    error,
    isSubscribed,
    refetch: fetchLogs,
  };
}
