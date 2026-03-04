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
}

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
	GenesisMessage string `json:",default=The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"`
}

// StorageConfig holds storage-related settings.
type StorageConfig struct {
	// DBPath is the file path for the bbolt database.
	DBPath string `json:",default=data/shitcoin.db"`
}
