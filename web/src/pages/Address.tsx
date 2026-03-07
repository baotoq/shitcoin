import { useState, useEffect } from "react";
import { useParams, Link } from "react-router";
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
import { fetchAddress } from "@/lib/api";
import type { AddressResponse } from "@/types/api";

const SATOSHI_PER_COIN = 100_000_000;

function formatCoins(satoshis: number): string {
  return (satoshis / SATOSHI_PER_COIN).toFixed(8);
}

export function Address() {
  const { addr } = useParams();
  const [data, setData] = useState<AddressResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    if (!addr) return;
    setIsLoading(true);
    setNotFound(false);
    fetchAddress(addr)
      .then((d) => setData(d))
      .catch(() => setNotFound(true))
      .finally(() => setIsLoading(false));
  }, [addr]);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="h-8 w-64 animate-pulse rounded bg-zinc-800" />
        <div className="h-48 animate-pulse rounded-xl bg-zinc-900" />
      </div>
    );
  }

  if (notFound || !data) {
    return (
      <div className="py-12 text-center">
        <h1 className="text-2xl font-bold text-zinc-100">Address not found</h1>
        <p className="mt-2 text-zinc-500">
          No data found for address {addr}.
        </p>
        <Link
          to="/"
          className="mt-4 inline-block text-blue-400 hover:underline"
        >
          Back to Dashboard
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-zinc-100">Address</h1>
        <p className="mt-1 break-all font-mono text-sm text-zinc-300">
          {data.address}
        </p>
      </div>

      {/* Balance */}
      <Card className="border-zinc-800 bg-zinc-900">
        <CardContent className="py-6 text-center">
          <p className="text-xs font-medium uppercase text-zinc-500">Balance</p>
          <p className="mt-1 text-4xl font-bold text-zinc-100">
            {formatCoins(data.balance)}
          </p>
          <p className="mt-1 text-sm text-zinc-500">coins</p>
        </CardContent>
      </Card>

      {/* UTXOs */}
      <Card className="border-zinc-800 bg-zinc-900">
        <CardHeader>
          <CardTitle className="text-zinc-100">
            UTXOs ({data.utxos.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          {data.utxos.length === 0 ? (
            <p className="py-4 text-center text-sm text-zinc-500">
              No UTXOs found for this address
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="border-zinc-800">
                  <TableHead className="text-zinc-400">TX ID</TableHead>
                  <TableHead className="text-zinc-400">Output Index</TableHead>
                  <TableHead className="text-right text-zinc-400">
                    Value
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.utxos.map((utxo) => (
                  <TableRow
                    key={`${utxo.txid}-${utxo.vout}`}
                    className="border-zinc-800 hover:bg-zinc-800/50"
                  >
                    <TableCell>
                      <Link
                        to={`/tx/${utxo.txid}`}
                        className="font-mono text-sm text-blue-400 hover:underline"
                      >
                        {utxo.txid.slice(0, 16)}...
                      </Link>
                    </TableCell>
                    <TableCell className="text-zinc-300">{utxo.vout}</TableCell>
                    <TableCell className="text-right font-mono text-zinc-300">
                      {formatCoins(utxo.value)}
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
