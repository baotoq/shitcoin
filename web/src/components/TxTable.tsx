import { Link } from "react-router";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import type { TxModel } from "@/types/api";

const SATOSHI_PER_COIN = 100_000_000;
const ZERO_HASH =
  "0000000000000000000000000000000000000000000000000000000000000000";

interface TxTableProps {
  transactions: TxModel[];
  showBlockContext?: boolean;
}

function isCoinbase(tx: TxModel): boolean {
  return (
    tx.inputs.length === 1 &&
    tx.inputs[0].txid === ZERO_HASH
  );
}

function totalOutputValue(tx: TxModel): number {
  return tx.outputs.reduce((sum, o) => sum + o.value, 0);
}

function formatCoins(satoshis: number): string {
  return (satoshis / SATOSHI_PER_COIN).toFixed(8);
}

export function TxTable({ transactions }: TxTableProps) {
  if (transactions.length === 0) {
    return (
      <p className="py-4 text-center text-sm text-zinc-500">
        No transactions
      </p>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow className="border-zinc-800">
          <TableHead className="text-zinc-400">TX ID</TableHead>
          <TableHead className="text-zinc-400">Inputs</TableHead>
          <TableHead className="text-zinc-400">Outputs</TableHead>
          <TableHead className="text-right text-zinc-400">
            Total Value
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {transactions.map((tx) => (
          <TableRow key={tx.id} className="border-zinc-800 hover:bg-zinc-800/50">
            <TableCell>
              <Link
                to={`/tx/${tx.id}`}
                className="font-mono text-sm text-blue-400 hover:underline"
              >
                {tx.id.slice(0, 16)}...
              </Link>
            </TableCell>
            <TableCell className="text-zinc-300">
              {isCoinbase(tx) ? (
                <Badge variant="secondary">Coinbase</Badge>
              ) : (
                tx.inputs.length
              )}
            </TableCell>
            <TableCell className="text-zinc-300">
              {tx.outputs.length}
            </TableCell>
            <TableCell className="text-right font-mono text-zinc-300">
              {formatCoins(totalOutputValue(tx))}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
