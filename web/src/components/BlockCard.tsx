import { useNavigate } from "react-router";
import { Card, CardContent } from "@/components/ui/card";
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

interface BlockCardProps {
  block: BlockModel;
  onClick?: () => void;
}

export function BlockCard({ block, onClick }: BlockCardProps) {
  const navigate = useNavigate();

  const handleClick = () => {
    if (onClick) {
      onClick();
    } else {
      navigate(`/blocks/${block.height}`);
    }
  };

  return (
    <Card
      className="cursor-pointer border-zinc-800 bg-zinc-900 transition-colors hover:bg-zinc-800"
      onClick={handleClick}
    >
      <CardContent className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-6">
          <span className="text-2xl font-bold text-zinc-100">
            #{block.height}
          </span>
          <span className="font-mono text-sm text-zinc-400">
            {block.hash.slice(0, 16)}...
          </span>
        </div>
        <div className="flex items-center gap-6 text-sm text-zinc-400">
          <span>{block.transactions.length} txs</span>
          <span>{timeAgo(block.header.timestamp)}</span>
        </div>
      </CardContent>
    </Card>
  );
}
