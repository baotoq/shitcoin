export interface StatusResponse {
  chain_height: number;
  latest_block_hash: string;
  mempool_size: number;
  peer_count: number;
  is_mining: boolean;
}

export interface BlockListResponse {
  blocks: BlockModel[];
  total: number;
  page: number;
  limit: number;
}

export interface BlockModel {
  hash: string;
  header: HeaderModel;
  height: number;
  message?: string;
  transactions: TxModel[];
}

export interface HeaderModel {
  version: number;
  prev_block_hash: string;
  merkle_root: string;
  timestamp: number;
  bits: number;
  nonce: number;
}

export interface TxModel {
  id: string;
  inputs: TxInputModel[];
  outputs: TxOutputModel[];
}

export interface TxInputModel {
  txid: string;
  vout: number;
  signature: string;
  pubkey: string;
}

export interface TxOutputModel {
  value: number;
  pubkey_hash: string;
  address: string;
}

export interface AddressResponse {
  address: string;
  balance: number;
  utxos: UTXOModel[];
}

export interface UTXOModel {
  txid: string;
  vout: number;
  value: number;
  pubkey_hash: string;
  address: string;
}

export interface SearchResult {
  type: string;
  block_height?: number;
  block_hash?: string;
  tx_hash?: string;
  address?: string;
}

export interface WSMessage {
  type: string;
  payload: unknown;
}

export interface MiningProgressPayload {
  nonce: number;
  hash: string;
  target: string;
  difficulty: number;
  block_height: number;
}

export interface ErrorResponse {
  error: string;
}
