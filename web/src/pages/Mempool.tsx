import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { TxTable } from "@/components/TxTable";
import { useWebSocket } from "@/hooks/useWebSocket";
import { fetchMempool } from "@/lib/api";
import type { TxModel } from "@/types/api";

export function Mempool() {
  const [txs, setTxs] = useState<TxModel[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const { lastMessage } = useWebSocket();

  const loadMempool = () => {
    fetchMempool()
      .then((data) => setTxs(data))
      .catch(() => {
        // Backend not available
      })
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadMempool();
  }, []);

  // Refresh on mempool_changed, new_tx, or new_block events
  useEffect(() => {
    if (
      lastMessage?.type === "mempool_changed" ||
      lastMessage?.type === "new_tx" ||
      lastMessage?.type === "new_block"
    ) {
      loadMempool();
    }
  }, [lastMessage]);

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-bold text-zinc-100">Mempool</h1>
        <Badge variant="outline">{txs.length} pending</Badge>
      </div>

      <Card className="border-zinc-800 bg-zinc-900">
        <CardHeader>
          <CardTitle className="text-zinc-100">Pending Transactions</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 3 }).map((_, i) => (
                <div
                  key={i}
                  className="h-10 animate-pulse rounded bg-zinc-800"
                />
              ))}
            </div>
          ) : txs.length === 0 ? (
            <div className="py-8 text-center">
              <p className="text-zinc-400">No pending transactions</p>
              <p className="mt-1 text-sm text-zinc-600">
                Transactions waiting to be included in a block will appear here.
              </p>
            </div>
          ) : (
            <TxTable transactions={txs} />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
