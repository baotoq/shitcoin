import { useState, useEffect } from "react";
import { Link } from "react-router";
import { Blocks, Users, Clock, Pickaxe } from "lucide-react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useNodeStatus } from "@/hooks/useNodeStatus";
import { useWebSocket } from "@/hooks/useWebSocket";
import { fetchBlocks } from "@/lib/api";
import type { BlockModel } from "@/types/api";

function timeAgo(timestamp: number): string {
  const seconds = Math.floor(Date.now() / 1000 - timestamp);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

function truncateHash(hash: string): string {
  if (hash.length <= 15) return hash;
  return hash.slice(0, 12) + "...";
}

const statCards = [
  { key: "chain_height" as const, label: "Chain Height", icon: Blocks },
  { key: "peer_count" as const, label: "Connected Peers", icon: Users },
  { key: "mempool_size" as const, label: "Mempool Size", icon: Clock },
] as const;

export function Dashboard() {
  const { status, isLoading } = useNodeStatus();
  const { lastMessage } = useWebSocket();
  const [recentBlocks, setRecentBlocks] = useState<BlockModel[]>([]);

  // Fetch recent blocks on mount
  useEffect(() => {
    fetchBlocks(1, 5)
      .then((res) => setRecentBlocks(res.blocks))
      .catch(() => {
        // Backend not available
      });
  }, []);

  // Refresh blocks on new_block WebSocket event
  useEffect(() => {
    if (lastMessage?.type === "new_block") {
      fetchBlocks(1, 5)
        .then((res) => setRecentBlocks(res.blocks))
        .catch(() => {
          // ignore
        });
    }
  }, [lastMessage]);

  return (
    <div className="space-y-6">
      {/* Stat cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {statCards.map((card) => (
          <Card key={card.key} className="border-zinc-800 bg-zinc-900">
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-zinc-400">
                {card.label}
              </CardTitle>
              <card.icon className="h-4 w-4 text-zinc-500" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-zinc-100">
                {isLoading || !status ? "---" : status[card.key]}
              </div>
            </CardContent>
          </Card>
        ))}

        {/* Mining status card */}
        <Card className="border-zinc-800 bg-zinc-900">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-zinc-400">
              Mining Status
            </CardTitle>
            <Pickaxe className="h-4 w-4 text-zinc-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {isLoading || !status ? (
                <span className="text-zinc-100">---</span>
              ) : status.is_mining ? (
                <span className="text-green-400">Active</span>
              ) : (
                <span className="text-zinc-500">Idle</span>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Recent Blocks */}
      <Card className="border-zinc-800 bg-zinc-900">
        <CardHeader>
          <CardTitle className="text-zinc-100">Recent Blocks</CardTitle>
        </CardHeader>
        <CardContent>
          {recentBlocks.length === 0 ? (
            <p className="text-sm text-zinc-500">
              {isLoading ? "Loading..." : "No blocks available"}
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="border-zinc-800">
                  <TableHead className="text-zinc-400">Height</TableHead>
                  <TableHead className="text-zinc-400">Hash</TableHead>
                  <TableHead className="text-zinc-400">Txs</TableHead>
                  <TableHead className="text-zinc-400">Time</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {recentBlocks.map((block) => (
                  <TableRow
                    key={block.hash}
                    className="border-zinc-800 hover:bg-zinc-800/50"
                  >
                    <TableCell>
                      <Link
                        to={`/blocks/${block.height}`}
                        className="font-mono text-blue-400 hover:underline"
                      >
                        {block.height}
                      </Link>
                    </TableCell>
                    <TableCell className="font-mono text-sm text-zinc-300">
                      {truncateHash(block.hash)}
                    </TableCell>
                    <TableCell className="text-zinc-300">
                      {block.transactions.length}
                    </TableCell>
                    <TableCell className="text-zinc-500">
                      {timeAgo(block.header.timestamp)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
