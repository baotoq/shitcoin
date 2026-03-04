package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestConfigLoading(t *testing.T) {
	// Write a full YAML config to a temp file
	yamlContent := `
Name: shitcoin
Host: 0.0.0.0
Port: 8080
Consensus:
  BlockTimeTarget: 10
  DifficultyAdjustInterval: 10
  InitialDifficulty: 16
  GenesisMessage: "Hello, Shitcoin!"
Storage:
  DBPath: data/shitcoin.db
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	var c Config
	conf.MustLoad(configPath, &c)

	// Verify rest config
	if c.Name != "shitcoin" {
		t.Errorf("Name = %q; want %q", c.Name, "shitcoin")
	}
	if c.Host != "0.0.0.0" {
		t.Errorf("Host = %q; want %q", c.Host, "0.0.0.0")
	}
	if c.Port != 8080 {
		t.Errorf("Port = %d; want %d", c.Port, 8080)
	}

	// Verify consensus config
	if c.Consensus.BlockTimeTarget != 10 {
		t.Errorf("BlockTimeTarget = %d; want %d", c.Consensus.BlockTimeTarget, 10)
	}
	if c.Consensus.DifficultyAdjustInterval != 10 {
		t.Errorf("DifficultyAdjustInterval = %d; want %d", c.Consensus.DifficultyAdjustInterval, 10)
	}
	if c.Consensus.InitialDifficulty != 16 {
		t.Errorf("InitialDifficulty = %d; want %d", c.Consensus.InitialDifficulty, 16)
	}
	if c.Consensus.GenesisMessage != "Hello, Shitcoin!" {
		t.Errorf("GenesisMessage = %q; want %q", c.Consensus.GenesisMessage, "Hello, Shitcoin!")
	}

	// Verify storage config
	if c.Storage.DBPath != "data/shitcoin.db" {
		t.Errorf("DBPath = %q; want %q", c.Storage.DBPath, "data/shitcoin.db")
	}
}

func TestConfigDefaults(t *testing.T) {
	// Write a minimal YAML config (just Name/Host/Port required by RestConf)
	yamlContent := `
Name: shitcoin
Host: 0.0.0.0
Port: 8080
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "minimal.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	var c Config
	conf.MustLoad(configPath, &c)
	c.Consensus.ApplyDefaults()

	// Verify defaults are applied for consensus
	if c.Consensus.BlockTimeTarget != 10 {
		t.Errorf("default BlockTimeTarget = %d; want %d", c.Consensus.BlockTimeTarget, 10)
	}
	if c.Consensus.DifficultyAdjustInterval != 10 {
		t.Errorf("default DifficultyAdjustInterval = %d; want %d", c.Consensus.DifficultyAdjustInterval, 10)
	}
	if c.Consensus.InitialDifficulty != 16 {
		t.Errorf("default InitialDifficulty = %d; want %d", c.Consensus.InitialDifficulty, 16)
	}
	expectedMsg := "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"
	if c.Consensus.GenesisMessage != expectedMsg {
		t.Errorf("default GenesisMessage = %q; want %q", c.Consensus.GenesisMessage, expectedMsg)
	}

	// Verify defaults for storage
	if c.Storage.DBPath != "data/shitcoin.db" {
		t.Errorf("default DBPath = %q; want %q", c.Storage.DBPath, "data/shitcoin.db")
	}
}
