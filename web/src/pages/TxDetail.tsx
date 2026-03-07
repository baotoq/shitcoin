import { useState, useEffect } from "react";
import { useParams, Link } from "react-router";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { TxModel } from "@/types/api";
import { fetchTx } from "@/lib/api";

const SATOSHI_PER_COIN = 100_000_000;
const ZERO_HASH =
  "0000000000000000000000000000000000000000000000000000000000000000";

function formatCoins(satoshis: number): string {
  return (satoshis / SATOSHI_PER_COIN).toFixed(8);
}

function isCoinbase(tx: TxModel): boolean {
  return tx.inputs.length === 1 && tx.inputs[0].txid === ZERO_HASH;
}

export function TxDetail() {
  const { hash } = useParams();
  const [tx, setTx] = useState<TxModel | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    if (!hash) return;
    setIsLoading(true);
    setNotFound(false);
    fetchTx(hash)
      .then((t) => setTx(t))
      .catch(() => setNotFound(true))
      .finally(() => setIsLoading(false));
  }, [hash]);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="h-8 w-64 animate-pulse rounded bg-zinc-800" />
        <div className="h-48 animate-pulse rounded-xl bg-zinc-900" />
      </div>
    );
  }

  if (notFound || !tx) {
    return (
      <div className="py-12 text-center">
        <h1 className="text-2xl font-bold text-zinc-100">
          Transaction not found
        </h1>
        <p className="mt-2 text-zinc-500">
          Transaction {hash} does not exist.
        </p>
        <Link
          to="/blocks"
          className="mt-4 inline-block text-blue-400 hover:underline"
        >
          Back to Block Explorer
        </Link>
      </div>
    );
  }

  const coinbase = isCoinbase(tx);

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-bold text-zinc-100">Transaction</h1>
        {coinbase && <Badge variant="secondary">Coinbase</Badge>}
      </div>

      {/* TX ID */}
      <Card className="border-zinc-800 bg-zinc-900">
        <CardContent className="pt-4">
          <p className="text-xs font-medium uppercase text-zinc-500">TX ID</p>
          <p className="mt-1 break-all font-mono text-sm text-zinc-200">
            {tx.id}
          </p>
        </CardContent>
      </Card>

      {/* Inputs */}
      <Card className="border-zinc-800 bg-zinc-900">
        <CardHeader>
          <CardTitle className="text-zinc-100">
            Inputs ({tx.inputs.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {tx.inputs.map((input, i) => (
              <div
                key={`${input.txid}-${input.vout}-${i}`}
                className="rounded-lg border border-zinc-800 bg-zinc-950 p-3"
              >
                <div className="flex flex-col gap-1">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-medium text-zinc-500">
                      TX ID:
                    </span>
                    {input.txid === ZERO_HASH ? (
                      <span className="font-mono text-xs text-zinc-500">
                        Coinbase (no input)
                      </span>
                    ) : (
                      <Link
                        to={`/tx/${input.txid}`}
                        className="font-mono text-xs text-blue-400 hover:underline"
                      >
                        {input.txid.slice(0, 16)}...
                      </Link>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-medium text-zinc-500">
                      Output Index:
                    </span>
                    <span className="text-xs text-zinc-300">{input.vout}</span>
                  </div>
                  {input.pubkey && input.txid !== ZERO_HASH && (
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-medium text-zinc-500">
                        PubKey:
                      </span>
                      <span className="font-mono text-xs text-zinc-400">
                        {input.pubkey.slice(0, 24)}...
                      </span>
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Outputs */}
      <Card className="border-zinc-800 bg-zinc-900">
        <CardHeader>
          <CardTitle className="text-zinc-100">
            Outputs ({tx.outputs.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {tx.outputs.map((output, i) => (
              <div
                key={`${output.address}-${i}`}
                className="flex items-center justify-between rounded-lg border border-zinc-800 bg-zinc-950 p-3"
              >
                <div className="flex flex-col gap-1">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-medium text-zinc-500">
                      Address:
                    </span>
                    <Link
                      to={`/address/${output.address}`}
                      className="font-mono text-xs text-blue-400 hover:underline"
                    >
                      {output.address}
                    </Link>
                  </div>
                </div>
                <span className="font-mono text-sm font-bold text-zinc-200">
                  {formatCoins(output.value)} coins
                </span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
