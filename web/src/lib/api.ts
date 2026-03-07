import type {
  StatusResponse,
  BlockListResponse,
  BlockModel,
  TxModel,
  AddressResponse,
  SearchResult,
} from "@/types/api";

async function fetchJSON<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error ${res.status}: ${text}`);
  }
  return res.json() as Promise<T>;
}

export function fetchStatus(): Promise<StatusResponse> {
  return fetchJSON<StatusResponse>("/api/status");
}

export function fetchBlocks(
  page: number = 1,
  limit: number = 20
): Promise<BlockListResponse> {
  return fetchJSON<BlockListResponse>(
    `/api/blocks?page=${page}&limit=${limit}`
  );
}

export function fetchBlockByHeight(height: number): Promise<BlockModel> {
  return fetchJSON<BlockModel>(`/api/blocks/${height}`);
}

export function fetchBlockByHash(hash: string): Promise<BlockModel> {
  return fetchJSON<BlockModel>(`/api/blocks/hash/${hash}`);
}

export function fetchTx(hash: string): Promise<TxModel> {
  return fetchJSON<TxModel>(`/api/tx/${hash}`);
}

export function fetchMempool(): Promise<TxModel[]> {
  return fetchJSON<TxModel[]>("/api/mempool");
}

export function fetchAddress(addr: string): Promise<AddressResponse> {
  return fetchJSON<AddressResponse>(`/api/address/${addr}`);
}

export function searchQuery(q: string): Promise<SearchResult> {
  return fetchJSON<SearchResult>(
    `/api/search?q=${encodeURIComponent(q)}`
  );
}
