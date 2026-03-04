package svc

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/baotoq/shitcoin/internal/config"
	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	boltrepo "github.com/baotoq/shitcoin/internal/infrastructure/persistence/bbolt"
	bolt "go.etcd.io/bbolt"
)

// ServiceContext holds all dependencies for the shitcoin node.
// Follows the go-zero ServiceContext pattern for dependency injection.
type ServiceContext struct {
	Config    config.Config
	ChainRepo chain.Repository
	Chain     *chain.Chain
	DB        *bolt.DB
}

// NewServiceContext creates a new ServiceContext by opening the bbolt database,
// creating the repository, and wiring up the Chain aggregate.
func NewServiceContext(c config.Config) *ServiceContext {
	// Ensure parent directories exist for DB path
	dbDir := filepath.Dir(c.Storage.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create db directory %s: %v", dbDir, err))
	}

	// Open bbolt database
	db, err := bolt.Open(c.Storage.DBPath, 0600, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to open bbolt database at %s: %v", c.Storage.DBPath, err))
	}

	// Create repository
	repo, err := boltrepo.NewBboltRepository(db)
	if err != nil {
		db.Close()
		panic(fmt.Sprintf("failed to create bbolt repository: %v", err))
	}

	// Create PoW service
	pow := &block.ProofOfWork{}

	// Create chain aggregate with config
	chainConfig := chain.ChainConfig{
		BlockTimeTarget:          c.Consensus.BlockTimeTarget,
		DifficultyAdjustInterval: c.Consensus.DifficultyAdjustInterval,
		InitialDifficulty:        c.Consensus.InitialDifficulty,
		GenesisMessage:           c.Consensus.GenesisMessage,
	}
	ch := chain.NewChain(repo, pow, chainConfig)

	return &ServiceContext{
		Config:    c,
		ChainRepo: repo,
		Chain:     ch,
		DB:        db,
	}
}

// Close releases resources held by the ServiceContext.
func (svc *ServiceContext) Close() {
	if svc.DB != nil {
		svc.DB.Close()
	}
}
