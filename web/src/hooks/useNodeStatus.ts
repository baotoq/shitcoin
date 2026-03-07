import { useState, useEffect, useCallback } from "react";
import type { StatusResponse } from "@/types/api";
import { fetchStatus } from "@/lib/api";
import { useWebSocket } from "@/hooks/useWebSocket";

export function useNodeStatus() {
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { lastMessage } = useWebSocket();

  const loadStatus = useCallback(async () => {
    try {
      const data = await fetchStatus();
      setStatus(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch status");
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Initial fetch
  useEffect(() => {
    loadStatus();
  }, [loadStatus]);

  // Poll every 10 seconds as fallback
  useEffect(() => {
    const interval = setInterval(loadStatus, 10000);
    return () => clearInterval(interval);
  }, [loadStatus]);

  // React to WebSocket messages
  useEffect(() => {
    if (!lastMessage) return;

    setStatus((prev) => {
      if (!prev) return prev;

      switch (lastMessage.type) {
        case "status":
          return lastMessage.payload as StatusResponse;

        case "new_block":
          return { ...prev, chain_height: prev.chain_height + 1 };

        case "mempool_changed": {
          const payload = lastMessage.payload as { size?: number };
          if (payload.size !== undefined) {
            return { ...prev, mempool_size: payload.size };
          }
          return prev;
        }

        case "peer_connected":
          return { ...prev, peer_count: prev.peer_count + 1 };

        case "peer_disconnected":
          return {
            ...prev,
            peer_count: Math.max(0, prev.peer_count - 1),
          };

        case "mining_started":
          return { ...prev, is_mining: true };

        case "mining_stopped":
          return { ...prev, is_mining: false };

        default:
          return prev;
      }
    });
  }, [lastMessage]);

  return { status, isLoading, error };
}
