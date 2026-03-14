package threshold

import "fmt"

func Check(coveragePct float64, minPct float64) error {
	if coveragePct < minPct {
		return fmt.Errorf("coverage %.1f%% below threshold %.1f%%", coveragePct, minPct)
	}
	return nil
}
