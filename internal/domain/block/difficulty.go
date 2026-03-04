package block

import (
	"math"
	"time"
)

// AdjustDifficulty computes new difficulty bits based on actual vs target block time.
//   - ratio < 1 (blocks too fast) increases bits (harder)
//   - ratio > 1 (blocks too slow) decreases bits (easier)
//   - Adjustment factor clamped to [0.25, 4.0]
//   - Result bits clamped to [1, 255]
func AdjustDifficulty(currentBits uint32, actualTimeSpan, targetTimeSpan time.Duration) uint32 {
	ratio := actualTimeSpan.Seconds() / targetTimeSpan.Seconds()

	// Clamp ratio to [0.25, 4.0]
	if ratio < 0.25 {
		ratio = 0.25
	}
	if ratio > 4.0 {
		ratio = 4.0
	}

	// Faster blocks (ratio < 1) -> divide by smaller number -> bits increase (harder)
	// Slower blocks (ratio > 1) -> divide by larger number -> bits decrease (easier)
	newBits := float64(currentBits) / ratio

	// Clamp result to [1, 255]
	result := uint32(math.Round(newBits))
	if result < 1 {
		result = 1
	}
	if result > 255 {
		result = 255
	}

	return result
}
