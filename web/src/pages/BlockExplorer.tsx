import { useState, useEffect, useCallback } from "react";
import { useSearchParams } from "react-router";
import { Button } from "@/components/ui/button";
import { BlockCard } from "@/components/BlockCard";
import { useWebSocket } from "@/hooks/useWebSocket";
import { fetchBlocks } from "@/lib/api";
import type { BlockModel } from "@/types/api";

const PAGE_SIZE = 20;

export function BlockExplorer() {
  const [searchParams, setSearchParams] = useSearchParams();
  const page = Number(searchParams.get("page") ?? "1");

  const [blocks, setBlocks] = useState<BlockModel[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const { lastMessage } = useWebSocket();

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const loadBlocks = useCallback(
    (p: number) => {
      setIsLoading(true);
      fetchBlocks(p, PAGE_SIZE)
        .then((res) => {
          setBlocks(res.blocks);
          setTotal(res.total);
        })
        .catch(() => {
          // Backend not available
        })
        .finally(() => setIsLoading(false));
    },
    []
  );

  useEffect(() => {
    loadBlocks(page);
  }, [page, loadBlocks]);

  // On new_block event, refetch if on page 1
  useEffect(() => {
    if (lastMessage?.type === "new_block" && page === 1) {
      loadBlocks(1);
    }
  }, [lastMessage, page, loadBlocks]);

  const goToPage = (p: number) => {
    setSearchParams({ page: String(p) });
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-zinc-100">Block Explorer</h1>
        <span className="text-sm text-zinc-400">{total} blocks total</span>
      </div>

      {isLoading && blocks.length === 0 ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <div
              key={i}
              className="h-16 animate-pulse rounded-xl bg-zinc-900"
            />
          ))}
        </div>
      ) : blocks.length === 0 ? (
        <p className="py-8 text-center text-zinc-500">No blocks available</p>
      ) : (
        <div className="space-y-2">
          {blocks.map((block) => (
            <BlockCard key={block.hash} block={block} />
          ))}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-4 pt-4">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => goToPage(page - 1)}
          >
            Previous
          </Button>
          <span className="text-sm text-zinc-400">
            Page {page} of {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= totalPages}
            onClick={() => goToPage(page + 1)}
          >
            Next
          </Button>
        </div>
      )}
    </div>
  );
}
