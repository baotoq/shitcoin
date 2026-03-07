import type { MiningProgressPayload } from "@/types/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface MiningVisualizerProps {
  progress: MiningProgressPayload | null;
  isActive: boolean;
  blockHeight?: number;
}

function formatNonce(n: number): string {
  return n.toLocaleString();
}

function highlightLeadingZeros(hash: string, target: string): React.ReactNode {
  // Count how many leading chars of target are '0'
  let matchLen = 0;
  for (let i = 0; i < target.length; i++) {
    if (target[i] === "0") {
      matchLen = i + 1;
    } else {
      break;
    }
  }

  if (matchLen === 0) {
    return <span>{hash}</span>;
  }

  return (
    <>
      <span className="text-green-400">{hash.slice(0, matchLen)}</span>
      <span>{hash.slice(matchLen)}</span>
    </>
  );
}

export function MiningVisualizer({
  progress,
  isActive,
  blockHeight,
}: MiningVisualizerProps) {
  if (!isActive) {
    return (
      <Card className="border-zinc-800 bg-zinc-900">
        <CardContent className="py-8 text-center">
          <p className="text-lg text-zinc-500">Mining is idle</p>
          <p className="mt-1 text-sm text-zinc-600">
            Start mining from the CLI to see real-time visualization
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="border-zinc-800 bg-zinc-900">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-zinc-100">
          <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-green-400" />
          Mining...
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Block Height */}
        <div>
          <p className="text-xs font-medium uppercase text-zinc-500">
            Block Height
          </p>
          <p className="text-3xl font-bold text-zinc-100">
            #{blockHeight ?? progress?.block_height ?? "---"}
          </p>
        </div>

        {progress ? (
          <>
            {/* Nonce */}
            <div>
              <p className="text-xs font-medium uppercase text-zinc-500">
                Current Nonce
              </p>
              <p className="font-mono text-xl text-zinc-100">
                {formatNonce(progress.nonce)}
              </p>
            </div>

            {/* Hash vs Target comparison */}
            <div className="space-y-2">
              <div>
                <p className="text-xs font-medium uppercase text-zinc-500">
                  Current Hash
                </p>
                <p className="break-all font-mono text-xs text-zinc-300">
                  {highlightLeadingZeros(progress.hash, progress.target)}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium uppercase text-zinc-500">
                  Target
                </p>
                <p className="break-all font-mono text-xs text-zinc-400">
                  {progress.target}
                </p>
              </div>
            </div>

            {/* Difficulty */}
            <div>
              <p className="text-xs font-medium uppercase text-zinc-500">
                Difficulty
              </p>
              <p className="text-sm text-zinc-300">{progress.difficulty}</p>
            </div>
          </>
        ) : (
          <p className="text-sm text-zinc-500">Waiting for mining data...</p>
        )}
      </CardContent>
    </Card>
  );
}
