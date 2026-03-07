import { useState } from "react";
import { useNavigate } from "react-router";
import { Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { searchQuery } from "@/lib/api";

export function SearchBar() {
  const [query, setQuery] = useState("");
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const q = query.trim();
    if (!q) return;

    setError(null);
    try {
      const result = await searchQuery(q);

      switch (result.type) {
        case "block":
        case "height":
          if (result.block_height !== undefined) {
            navigate(`/blocks/${result.block_height}`);
          }
          break;
        case "tx":
          if (result.tx_hash) {
            navigate(`/tx/${result.tx_hash}`);
          }
          break;
        case "address":
          if (result.address) {
            navigate(`/address/${result.address}`);
          }
          break;
        default:
          setError("Unknown result type");
      }
    } catch {
      setError("Not found");
    }
  }

  return (
    <div className="border-b border-zinc-800 bg-zinc-900/50 px-4 py-2">
      <form onSubmit={handleSubmit} className="mx-auto max-w-[600px]">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-500" />
          <Input
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setError(null);
            }}
            placeholder="Search by block hash, tx hash, height, or address..."
            className="border-zinc-700 bg-zinc-800 pl-10 text-zinc-200 placeholder:text-zinc-500"
          />
        </div>
        {error && (
          <p className="mt-1 text-center text-xs text-red-400">{error}</p>
        )}
      </form>
    </div>
  );
}
