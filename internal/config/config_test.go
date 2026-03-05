package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	var c Config
	conf.MustLoad(configPath, &c)

	// Verify rest config
	assert.Equal(t, "shitcoin", c.Name)
	assert.Equal(t, "0.0.0.0", c.Host)
	assert.Equal(t, 8080, c.Port)

	// Verify consensus config
	assert.Equal(t, 10, c.Consensus.BlockTimeTarget)
	assert.Equal(t, 10, c.Consensus.DifficultyAdjustInterval)
	assert.Equal(t, 16, c.Consensus.InitialDifficulty)
	assert.Equal(t, "Hello, Shitcoin!", c.Consensus.GenesisMessage)

	// Verify storage config
	assert.Equal(t, "data/shitcoin.db", c.Storage.DBPath)
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
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	var c Config
	conf.MustLoad(configPath, &c)
	c.Consensus.ApplyDefaults()

	// Verify defaults are applied for consensus
	assert.Equal(t, 10, c.Consensus.BlockTimeTarget)
	assert.Equal(t, 10, c.Consensus.DifficultyAdjustInterval)
	assert.Equal(t, 16, c.Consensus.InitialDifficulty)
	expectedMsg := "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"
	assert.Equal(t, expectedMsg, c.Consensus.GenesisMessage)

	// Verify defaults for storage
	assert.Equal(t, "data/shitcoin.db", c.Storage.DBPath)
}
