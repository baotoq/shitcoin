import { Blocks, Users, Clock, Circle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { useNodeStatus } from "@/hooks/useNodeStatus";

export function StatusBar() {
  const { status, isLoading } = useNodeStatus();

  return (
    <div className="flex h-12 items-center justify-between border-b border-zinc-800 bg-zinc-900 px-4 text-white">
      <span className="text-sm font-semibold tracking-wide">
        Shitcoin Explorer
      </span>

      <div className="flex items-center gap-3">
        {isLoading || !status ? (
          <span className="text-xs text-zinc-500">Loading...</span>
        ) : (
          <>
            <Badge
              variant="secondary"
              className="flex items-center gap-1.5 bg-zinc-800 text-zinc-300"
            >
              <Blocks className="h-3 w-3" />
              {status.chain_height}
            </Badge>

            <Badge
              variant="secondary"
              className="flex items-center gap-1.5 bg-zinc-800 text-zinc-300"
            >
              <Users className="h-3 w-3" />
              {status.peer_count}
            </Badge>

            <Badge
              variant="secondary"
              className="flex items-center gap-1.5 bg-zinc-800 text-zinc-300"
            >
              <Clock className="h-3 w-3" />
              {status.mempool_size}
            </Badge>

            <Badge
              variant="secondary"
              className={`flex items-center gap-1.5 ${
                status.is_mining
                  ? "bg-green-900/50 text-green-400"
                  : "bg-zinc-800 text-zinc-500"
              }`}
            >
              <Circle
                className={`h-2 w-2 fill-current ${
                  status.is_mining ? "text-green-400" : "text-zinc-500"
                }`}
              />
              {status.is_mining ? "Mining" : "Idle"}
            </Badge>
          </>
        )}
      </div>
    </div>
  );
}
