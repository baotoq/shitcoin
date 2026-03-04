package block

import "time"

// AdjustDifficulty computes new difficulty bits based on actual vs target block time.
// - ratio < 1 (blocks too fast) increases bits (harder)
// - ratio > 1 (blocks too slow) decreases bits (easier)
// - Adjustment factor clamped to [0.25, 4.0]
// - Result bits clamped to [1, 255]
func AdjustDifficulty(currentBits uint32, actualTimeSpan, targetTimeSpan time.Duration) uint32 {
	panic("not implemented")
}
