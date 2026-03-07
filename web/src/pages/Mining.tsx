import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { MiningVisualizer } from "@/components/MiningVisualizer";
import { BlockCard } from "@/components/BlockCard";
import { useWebSocket } from "@/hooks/useWebSocket";
import { fetchStatus, fetchBlockByHeight } from "@/lib/api";
import type { MiningProgressPayload, BlockModel } from "@/types/api";

export function Mining() {
  const [isActive, setIsActive] = useState(false);
  const [progress, setProgress] = useState<MiningProgressPayload | null>(null);
  const [blockHeight, setBlockHeight] = useState<number | undefined>();
  const [lastMinedBlock, setLastMinedBlock] = useState<BlockModel | null>(null);
  const { lastMessage } = useWebSocket();

  // Fetch initial mining status
  useEffect(() => {
    fetchStatus()
      .then((s) => {
        setIsActive(s.is_mining);
      })
      .catch(() => {
        // Backend not available
      });
  }, []);

  // Handle WebSocket mining events
  useEffect(() => {
    if (!lastMessage) return;

    switch (lastMessage.type) {
      case "mining_started": {
        setIsActive(true);
        setProgress(null);
        const payload = lastMessage.payload as { block_height?: number };
        if (payload?.block_height !== undefined) {
          setBlockHeight(payload.block_height);
        }
        break;
      }
      case "mining_progress": {
        const p = lastMessage.payload as MiningProgressPayload;
        setProgress(p);
        if (p.block_height !== undefined) {
          setBlockHeight(p.block_height);
        }
        break;
      }
      case "mining_stopped": {
        setIsActive(false);
        setProgress(null);
        // Fetch the last mined block
        const stoppedPayload = lastMessage.payload as {
          block_height?: number;
        };
        if (stoppedPayload?.block_height !== undefined) {
          fetchBlockByHeight(stoppedPayload.block_height)
            .then((b) => setLastMinedBlock(b))
            .catch(() => {
              // ignore
            });
        }
        break;
      }
      case "new_block": {
        // A new block was mined/received; fetch it as the last mined block
        const blockPayload = lastMessage.payload as { height?: number };
        if (blockPayload?.height !== undefined) {
          fetchBlockByHeight(blockPayload.height)
            .then((b) => setLastMinedBlock(b))
            .catch(() => {
              // ignore
            });
        }
        break;
      }
    }
  }, [lastMessage]);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-zinc-100">Mining</h1>

      <MiningVisualizer
        progress={progress}
        isActive={isActive}
        blockHeight={blockHeight}
      />

      {/* Educational note */}
      <Card className="border-zinc-800 bg-zinc-900">
        <CardContent className="py-4">
          <p className="text-sm text-zinc-400">
            Watch the miner search for a nonce that produces a hash below the
            target value. The hash must have enough leading zeros to be less than
            the target -- this is what makes mining computationally expensive and
            secures the blockchain.
          </p>
        </CardContent>
      </Card>

      {/* Last mined block */}
      {lastMinedBlock && (
        <Card className="border-zinc-800 bg-zinc-900">
          <CardHeader>
            <CardTitle className="text-zinc-100">Last Mined Block</CardTitle>
          </CardHeader>
          <CardContent>
            <BlockCard block={lastMinedBlock} />
          </CardContent>
        </Card>
      )}
    </div>
  );
}
