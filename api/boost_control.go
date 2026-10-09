package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
)

var (
	errBoostConflict = errors.New("BOOST state conflict")
	errBoostInvalid = errors.New("invalid BOOST number")
)

type boostSnapshot struct {
	Power bool
	Mask uint32
	Remaining [2]int64
}

type boostDevice interface {
	ReadBoost(context.Context) (boostSnapshot, error)
	WriteBoostMask(context.Context, uint32) error
}

type boostResult struct {
	OK bool `json:"ok"`
	Boost string `json:"boost"`
	Active bool `json:"active"`
	Changed bool `json:"changed"`
	Verified bool `json:"verified"`
	StateMask uint32 `json:"state_mask"`
	RemainingMinutes *int64 `json:"remaining_minutes"`
}

// decodeBoostSnapshot rejects missing/null values instead of treating them as OFF.
func decodeBoostSnapshot(raw []byte) (boostSnapshot, error) {
	var values map[string][]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return boostSnapshot{}, fmt.Errorf("invalid BOOST values: %w", err)
	}
	readInt := func(key string, min, max float64) (int64, error) {
		v := values[key]
		if len(v) == 0 || string(v[0]) == "null" {
			return 0, fmt.Errorf("BOOST parameter %s is missing or null", key)
		}
		var number float64
		if err := json.Unmarshal(v[0], &number); err != nil {
			return 0, fmt.Errorf("invalid BOOST parameter %s: %w", key, err)
		}
		if math.IsNaN(number) || math.IsInf(number, 0) || number != math.Trunc(number) || number < min || number > max {
			return 0, fmt.Errorf("BOOST parameter %s is out of range", key)
		}
		return int64(number), nil
	}
	power, err := readInt("u7074", 0, 1)
	if err != nil { return boostSnapshot{}, err }
	mask, err := readInt("u6639", 0, math.MaxUint32)
	if err != nil { return boostSnapshot{}, err }
	first, err := readInt("u7427", -1, math.MaxInt32)
	if err != nil { return boostSnapshot{}, err }
	second, err := readInt("u7428", -1, math.MaxInt32)
	if err != nil { return boostSnapshot{}, err }
	return boostSnapshot{Power: power == 1, Mask: uint32(mask), Remaining: [2]int64{first, second}}, nil
}

func planBoostChange(before boostSnapshot, boost int, on bool) (uint32, bool, error) {
	if boost != 1 && boost != 2 { return 0, false, errBoostInvalid }
	flag := uint32(64) << (boost - 1)
	index := boost - 1
	if !on {
		if before.Mask&flag == 0 {
			if before.Remaining[index] >= 0 {
				return 0, false, fmt.Errorf("%w: BOOST %d flag and timer disagree; refresh state", errBoostConflict, boost)
			}
			return before.Mask, false, nil
		}
		// Never reset the whole register: retain the other BOOST and all other bits.
		return before.Mask &^ flag, true, nil
	}
	if !before.Power {
		return 0, false, fmt.Errorf("%w: ventilation must be on", errBoostConflict)
	}
	// Repeating ON on an already running BOOST must not restart its timer.
	if before.Mask == flag && before.Remaining[index] >= 0 && before.Remaining[1-index] == -1 {
		return before.Mask, false, nil
	}
	if before.Mask != 0 || before.Remaining[0] != -1 || before.Remaining[1] != -1 {
		return 0, false, fmt.Errorf("%w: another timed mode is active or state is inconsistent", errBoostConflict)
	}
	return before.Mask | flag, true, nil
}

func boostStateMatches(before, after boostSnapshot, boost int, on bool) bool {
	flag := uint32(64) << (boost - 1)
	if after.Power != before.Power || after.Mask &^ flag != before.Mask &^ flag {
		return false
	}
	if on { return after.Mask&flag != 0 && after.Remaining[boost-1] >= 0 }
	return after.Mask&flag == 0 && after.Remaining[boost-1] == -1
}

// Exactly one write; only readback is retried. The caller serializes commands.
func changeBoost(ctx context.Context, device boostDevice, boost int, on bool, interval time.Duration, attempts int) (boostResult, error) {
	if boost != 1 && boost != 2 { return boostResult{}, errBoostInvalid }
	if err := ctx.Err(); err != nil { return boostResult{}, err }
	before, err := device.ReadBoost(ctx)
	if err != nil { return boostResult{}, err }
	mask, changed, err := planBoostChange(before, boost, on)
	if err != nil { return boostResult{}, err }
	result := func(state boostSnapshot) boostResult {
		r := boostResult{OK: true, Boost: fmt.Sprint(boost), Active: on, Changed: changed, Verified: true, StateMask: state.Mask}
		if on { minutes := state.Remaining[boost-1]; r.RemainingMinutes = &minutes }
		return r
	}
	if !changed { return result(before), nil }
	if err := ctx.Err(); err != nil { return boostResult{}, err }
	if err := device.WriteBoostMask(ctx, mask); err != nil {
		return boostResult{}, fmt.Errorf("BOOST command failed; refresh state before retrying: %w", err)
	}
	if attempts < 1 { attempts = 1 }
	var readErr error
	for i := 0; i < attempts; i++ {
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return boostResult{}, fmt.Errorf("BOOST command sent but state not verified: %w", ctx.Err())
		case <-timer.C:
		}
		after, err := device.ReadBoost(ctx)
		readErr = err
		if err == nil && boostStateMatches(before, after, boost, on) { return result(after), nil }
	}
	if readErr != nil { return boostResult{}, fmt.Errorf("BOOST command acknowledged but readback failed: %w", readErr) }
	return boostResult{}, errors.New("BOOST command acknowledged but requested flag/timer state not confirmed; refresh state")
}
