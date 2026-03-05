package block

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAdjustDifficulty(t *testing.T) {
	tests := []struct {
		name           string
		currentBits    uint32
		actualTimeSpan time.Duration
		targetTimeSpan time.Duration
		wantHigher     bool   // expect bits to increase (harder)
		wantLower      bool   // expect bits to decrease (easier)
		wantEqual      bool   // expect bits unchanged
		wantExact      uint32 // exact expected value (0 means don't check exact)
	}{
		{
			name:           "blocks too fast - difficulty increases",
			currentBits:    16,
			actualTimeSpan: 5 * time.Second,
			targetTimeSpan: 10 * time.Second,
			wantHigher:     true,
		},
		{
			name:           "blocks too slow - difficulty decreases",
			currentBits:    16,
			actualTimeSpan: 20 * time.Second,
			targetTimeSpan: 10 * time.Second,
			wantLower:      true,
		},
		{
			name:           "blocks on target - difficulty unchanged",
			currentBits:    16,
			actualTimeSpan: 10 * time.Second,
			targetTimeSpan: 10 * time.Second,
			wantEqual:      true,
			wantExact:      16,
		},
		{
			name:           "clamp max - extremely fast blocks capped at 4x",
			currentBits:    16,
			actualTimeSpan: 100 * time.Millisecond, // 100x faster
			targetTimeSpan: 10 * time.Second,
			wantExact:      64, // 16 * 4 = 64
		},
		{
			name:           "clamp min - extremely slow blocks capped at 0.25x",
			currentBits:    16,
			actualTimeSpan: 100 * time.Second, // 10x slower
			targetTimeSpan: 10 * time.Second,
			wantExact:      4, // 16 * 0.25 = 4
		},
		{
			name:           "clamp bits range low - result never below 1",
			currentBits:    1,
			actualTimeSpan: 40 * time.Second,
			targetTimeSpan: 10 * time.Second,
			wantExact:      1, // 1 / 4 = 0.25 -> clamped to 1
		},
		{
			name:           "clamp bits range high - result never above 255",
			currentBits:    200,
			actualTimeSpan: 1 * time.Second,
			targetTimeSpan: 10 * time.Second,
			wantHigher:     true,
		},
		{
			name:           "high bits clamped to 255 max",
			currentBits:    250,
			actualTimeSpan: 1 * time.Second, // very fast -> 4x clamp
			targetTimeSpan: 10 * time.Second,
			wantExact:      255, // 250 * 4 = 1000, clamped to 255
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			got := AdjustDifficulty(tt.currentBits, tt.actualTimeSpan, tt.targetTimeSpan)

			if tt.wantExact != 0 {
				assert.Equal(tt.wantExact, got)
			}

			if tt.wantHigher {
				assert.Greater(got, tt.currentBits)
			}

			if tt.wantLower {
				assert.Less(got, tt.currentBits)
			}

			if tt.wantEqual {
				assert.Equal(tt.currentBits, got)
			}

			// Invariant: result always in [1, 255]
			assert.GreaterOrEqual(got, uint32(1))
			assert.LessOrEqual(got, uint32(255))
		})
	}
}
