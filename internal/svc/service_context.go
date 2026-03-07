package svc

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/baotoq/shitcoin/internal/config"
	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/events"
	"github.com/baotoq/shitcoin/internal/domain/mempool"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/baotoq/shitcoin/internal/domain/wallet"
	boltrepo "github.com/baotoq/shitcoin/internal/infrastructure/persistence/bbolt"
	"github.com/baotoq/shitcoin/internal/infrastructure/persistence/jsonfile"
	bolt "go.etcd.io/bbolt"
)

// ServiceContext holds all dependencies for the shitcoin node.
// Follows the go-zero ServiceContext pattern for dependency injection.
type ServiceContext struct {
	Config     config.Config
	ChainRepo  chain.Repository
	Chain      *chain.Chain
	UTXORepo   utxo.Repository
	UTXOSet    *utxo.Set
	WalletRepo wallet.Repository
	Mempool    *mempool.Mempool
	EventBus   *events.Bus
	DB         *bolt.DB
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

	// Create chain repository
	repo, err := boltrepo.NewBboltRepository(db)
	if err != nil {
		db.Close()
		panic(fmt.Sprintf("failed to create bbolt repository: %v", err))
	}

	// Create UTXO repository and set
	utxoRepo, err := boltrepo.NewUTXORepo(db)
	if err != nil {
		db.Close()
		panic(fmt.Sprintf("failed to create utxo repository: %v", err))
	}
	utxoSet := utxo.NewSet(utxoRepo)

	// Create wallet repository
	walletRepo, err := jsonfile.NewWalletRepo(c.Storage.WalletPath)
	if err != nil {
		db.Close()
		panic(fmt.Sprintf("failed to create wallet repository: %v", err))
	}

	// Create mempool
	pool := mempool.New(utxoSet)

	// Create PoW service
	pow := &block.ProofOfWork{}

	// Create chain aggregate with config
	chainConfig := chain.ChainConfig{
		BlockTimeTarget:          c.Consensus.BlockTimeTarget,
		DifficultyAdjustInterval: c.Consensus.DifficultyAdjustInterval,
		InitialDifficulty:        c.Consensus.InitialDifficulty,
		GenesisMessage:           c.Consensus.GenesisMessage,
		BlockReward:              c.Consensus.BlockReward,
		HalvingInterval:          c.Consensus.HalvingInterval,
		MaxBlockTxs:              c.Consensus.MaxBlockTxs,
	}
	ch := chain.NewChain(repo, pow, chainConfig, utxoSet)

	return &ServiceContext{
		Config:     c,
		ChainRepo:  repo,
		Chain:      ch,
		UTXORepo:   utxoRepo,
		UTXOSet:    utxoSet,
		WalletRepo: walletRepo,
		Mempool:    pool,
		EventBus:   events.NewBus(),
		DB:         db,
	}
}

// Close releases resources held by the ServiceContext.
func (svc *ServiceContext) Close() {
	if svc.DB != nil {
		svc.DB.Close()
	}
}
