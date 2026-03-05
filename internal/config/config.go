package config

import "github.com/zeromicro/go-zero/rest"

// Config is the root configuration struct for the shitcoin node.
// It embeds go-zero's RestConf for HTTP server settings and adds
// blockchain-specific configuration sections.
//
// IMPORTANT: go-zero uses `json` struct tags for ALL config formats
// (YAML, JSON, TOML), not `yaml` tags. See Pitfall #7 in research.
type Config struct {
	rest.RestConf
	Consensus ConsensusConfig `json:"Consensus"`
	Storage   StorageConfig   `json:"Storage"`
	P2P       P2PConfig       `json:"P2P"`
}

// SatoshiPerCoin is the number of satoshis in one coin.
const SatoshiPerCoin int64 = 100_000_000

// ConsensusConfig holds blockchain consensus parameters.
// All fields have defaults so a minimal config (just Name/Host/Port) works.
type ConsensusConfig struct {
	// BlockTimeTarget is the target number of seconds between blocks.
	BlockTimeTarget int `json:",default=10"`

	// DifficultyAdjustInterval is how many blocks between difficulty adjustments.
	DifficultyAdjustInterval int `json:",default=10"`

	// InitialDifficulty is the initial number of leading zero bits required in a block hash.
	InitialDifficulty int `json:",default=16"`

	// GenesisMessage is the message embedded in the genesis block.
	// Default is applied in ApplyDefaults() to avoid go vet warning about spaces in struct tags.
	GenesisMessage string `json:",optional"`

	// BlockReward is the coinbase reward in satoshis (default: 50 coins = 5,000,000,000 satoshis).
	BlockReward int64 `json:",default=5000000000"`
}

// DefaultGenesisMessage is the default genesis block message when none is configured.
const DefaultGenesisMessage = "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"

// ApplyDefaults fills in zero-value fields with their default values.
// Call this after conf.MustLoad to handle defaults that cannot be expressed
// in struct tags (e.g., strings with spaces).
func (c *ConsensusConfig) ApplyDefaults() {
	if c.GenesisMessage == "" {
		c.GenesisMessage = DefaultGenesisMessage
	}
}

// P2PConfig holds peer-to-peer networking settings.
type P2PConfig struct {
	// Port is the TCP port for the P2P server.
	Port int `json:",default=3000"`
	// Peers is a comma-separated list of seed peer addresses (host:port).
	Peers string `json:",optional"`
}

// StorageConfig holds storage-related settings.
type StorageConfig struct {
	// DBPath is the file path for the bbolt database.
	DBPath string `json:",default=data/shitcoin.db"`
	// WalletPath is the file path for the JSON wallet file.
	WalletPath string `json:",default=data/wallets.json"`
}
