import { useState, useEffect } from "react";
import { useParams, Link } from "react-router";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { TxTable } from "@/components/TxTable";
import { fetchBlockByHeight } from "@/lib/api";
import type { BlockModel } from "@/types/api";

function formatTimestamp(ts: number): string {
  return new Date(ts * 1000).toLocaleString();
}

export function BlockDetail() {
  const { height } = useParams();
  const heightNum = Number(height);

  const [block, setBlock] = useState<BlockModel | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    setIsLoading(true);
    setNotFound(false);
    fetchBlockByHeight(heightNum)
      .then((b) => setBlock(b))
      .catch(() => setNotFound(true))
      .finally(() => setIsLoading(false));
  }, [heightNum]);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="h-8 w-48 animate-pulse rounded bg-zinc-800" />
        <div className="h-64 animate-pulse rounded-xl bg-zinc-900" />
      </div>
    );
  }

  if (notFound || !block) {
    return (
      <div className="py-12 text-center">
        <h1 className="text-2xl font-bold text-zinc-100">Block not found</h1>
        <p className="mt-2 text-zinc-500">
          Block at height {height} does not exist.
        </p>
        <Link to="/blocks" className="mt-4 inline-block text-blue-400 hover:underline">
          Back to Block Explorer
        </Link>
      </div>
    );
  }

  const details = [
    { label: "Height", value: block.height },
    {
      label: "Hash",
      value: (
        <span className="break-all font-mono text-xs">{block.hash}</span>
      ),
    },
    {
      label: "Previous Block",
      value:
        block.height > 0 ? (
          <Link
            to={`/blocks/${block.height - 1}`}
            className="break-all font-mono text-xs text-blue-400 hover:underline"
          >
            {block.header.prev_block_hash}
          </Link>
        ) : (
          <span className="font-mono text-xs text-zinc-500">Genesis</span>
        ),
    },
    {
      label: "Merkle Root",
      value: (
        <span className="break-all font-mono text-xs">
          {block.header.merkle_root}
        </span>
      ),
    },
    { label: "Timestamp", value: formatTimestamp(block.header.timestamp) },
    { label: "Bits (Difficulty)", value: block.header.bits },
    {
      label: "Nonce",
      value: block.header.nonce.toLocaleString(),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-zinc-100">
          Block #{block.height}
        </h1>
        <div className="flex gap-2">
          {block.height > 0 && (
            <Link to={`/blocks/${block.height - 1}`}>
              <Button variant="outline" size="sm">
                Previous Block
              </Button>
            </Link>
          )}
          <Link to={`/blocks/${block.height + 1}`}>
            <Button variant="outline" size="sm">
              Next Block
            </Button>
          </Link>
        </div>
      </div>

      {/* Block details */}
      <Card className="border-zinc-800 bg-zinc-900">
        <CardHeader>
          <CardTitle className="text-zinc-100">Block Header</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="space-y-3">
            {details.map((d) => (
              <div key={d.label} className="flex flex-col gap-0.5 sm:flex-row sm:gap-4">
                <dt className="w-36 shrink-0 text-sm font-medium text-zinc-400">
                  {d.label}
                </dt>
                <dd className="text-sm text-zinc-200">{d.value}</dd>
              </div>
            ))}
            {block.message && (
              <div className="flex flex-col gap-0.5 sm:flex-row sm:gap-4">
                <dt className="w-36 shrink-0 text-sm font-medium text-zinc-400">
                  Message
                </dt>
                <dd className="text-sm italic text-zinc-300">
                  {block.message}
                </dd>
              </div>
            )}
          </dl>
        </CardContent>
      </Card>

      {/* Transactions */}
      <Card className="border-zinc-800 bg-zinc-900">
        <CardHeader>
          <CardTitle className="text-zinc-100">
            Transactions ({block.transactions.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          <TxTable transactions={block.transactions} />
        </CardContent>
      </Card>
    </div>
  );
}
