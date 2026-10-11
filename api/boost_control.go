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
    errBoostConflict = errors.New("timed mode state conflict")
    errBoostInvalid  = errors.New("invalid timed mode")
)

const (
    boost1Flag    uint32 = 64
    boost2Flag    uint32 = 128
    fireplaceFlag uint32 = 256
)

// Snapshot of one controller: all three timers are required for safe writes.
// -1 means inactive; a non-negative number means an active countdown.
type boostSnapshot struct {
    Power     bool
    Mask      uint32
    Remaining [3]int64
}

type boostDevice interface {
    ReadBoost(context.Context) (boostSnapshot, error)
    WriteBoostMask(context.Context, uint32) error
}

type boostResult struct {
    OK               bool   `json:"ok"`
    Boost            string `json:"boost"`
    Active           bool   `json:"active"`
    Changed          bool   `json:"changed"`
    Verified         bool   `json:"verified"`
    StateMask        uint32 `json:"state_mask"`
    RemainingMinutes *int64 `json:"remaining_minutes"`
}

type fireplaceResult struct {
    OK               bool   `json:"ok"`
    Mode             string `json:"mode"`
    Active           bool   `json:"active"`
    Changed          bool   `json:"changed"`
    Verified         bool   `json:"verified"`
    StateMask        uint32 `json:"state_mask"`
    RemainingMinutes *int64 `json:"remaining_minutes"`
}

type timedResult struct {
    OK               bool
    Active           bool
    Changed          bool
    Verified         bool
    StateMask        uint32
    RemainingMinutes *int64
}

func timedFlag(mode int) (uint32, error) {
    switch mode {
    case 1:
        return boost1Flag, nil
    case 2:
        return boost2Flag, nil
    case 3:
        return fireplaceFlag, nil
    default:
        return 0, errBoostInvalid
    }
}

// decodeBoostSnapshot refuses missing, null, fractional, or malformed timer
// readings; unknown state must never silently be interpreted as OFF.
func decodeBoostSnapshot(raw []byte) (boostSnapshot, error) {
    var values map[string][]json.RawMessage
    if err := json.Unmarshal(raw, &values); err != nil {
        return boostSnapshot{}, fmt.Errorf("invalid timed mode values: %w", err)
    }
    readInt := func(key string, min, max float64) (int64, error) {
        value := values[key]
        if len(value) == 0 || string(value[0]) == "null" {
            return 0, fmt.Errorf("timed mode parameter %s is missing or null", key)
        }
        var number float64
        if err := json.Unmarshal(value[0], &number); err != nil {
            return 0, fmt.Errorf("invalid timed mode parameter %s: %w", key, err)
        }
        if math.IsNaN(number) || math.IsInf(number, 0) ||
            number != math.Trunc(number) || number < min || number > max {
            return 0, fmt.Errorf("timed mode parameter %s is out of range", key)
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
    third, err := readInt("u7429", -1, math.MaxInt32)
    if err != nil { return boostSnapshot{}, err }
    return boostSnapshot{Power: power == 1, Mask: uint32(mask), Remaining: [3]int64{first, second, third}}, nil
}

// Start only when no other timed mode is running. Stop clears ONLY the
// requested bit, preserving all other flags even for anomalous overlap.
func planTimedChange(before boostSnapshot, mode int, on bool) (uint32, bool, error) {
    flag, err := timedFlag(mode)
    if err != nil { return 0, false, err }
    index := mode - 1
    if !on {
        if before.Mask&flag == 0 {
            if before.Remaining[index] >= 0 {
                return 0, false, fmt.Errorf("%w: mode %d flag and timer disagree; refresh state", errBoostConflict, mode)
            }
            return before.Mask, false, nil
        }
        return before.Mask &^ flag, true, nil
    }
    if !before.Power {
        return 0, false, fmt.Errorf("%w: ventilation must be on", errBoostConflict)
    }
    if before.Mask == flag && before.Remaining[index] >= 0 {
        othersIdle := true
        for i, remaining := range before.Remaining {
            if i != index && remaining != -1 { othersIdle = false }
        }
        if othersIdle { return before.Mask, false, nil }
    }
    if before.Mask != 0 {
        return 0, false, fmt.Errorf("%w: another timed mode is active", errBoostConflict)
    }
    for _, remaining := range before.Remaining {
        if remaining != -1 {
            return 0, false, fmt.Errorf("%w: timer active without mask", errBoostConflict)
        }
    }
    return before.Mask | flag, true, nil
}

func planBoostChange(before boostSnapshot, boost int, on bool) (uint32, bool, error) {
    if boost != 1 && boost != 2 { return 0, false, errBoostInvalid }
    return planTimedChange(before, boost, on)
}

func planFireplaceChange(before boostSnapshot, on bool) (uint32, bool, error) {
    return planTimedChange(before, 3, on)
}

func timedStateMatches(before, after boostSnapshot, mode int, on bool) bool {
    flag, err := timedFlag(mode)
    if err != nil { return false }
    if after.Power != before.Power || after.Mask &^ flag != before.Mask &^ flag {
        return false
    }
    if on {
        return after.Mask&flag != 0 && after.Remaining[mode-1] >= 0
    }
    return after.Mask&flag == 0 && after.Remaining[mode-1] == -1
}

func boostStateMatches(before, after boostSnapshot, boost int, on bool) bool {
    if boost != 1 && boost != 2 { return false }
    return timedStateMatches(before, after, boost, on)
}

// Exactly one write; retry only the state readback. Caller serializes
// all timed-mode commands using the same mutex, including legacy writes.
func changeTimed(ctx context.Context, device boostDevice, mode int, on bool, interval time.Duration, attempts int) (timedResult, error) {
    if _, err := timedFlag(mode); err != nil { return timedResult{}, err }
    if err := ctx.Err(); err != nil { return timedResult{}, err }
    before, err := device.ReadBoost(ctx)
    if err != nil { return timedResult{}, err }
    mask, changed, err := planTimedChange(before, mode, on)
    if err != nil { return timedResult{}, err }
    result := func(state boostSnapshot) timedResult {
        r := timedResult{OK: true, Active: on, Changed: changed, Verified: true, StateMask: state.Mask}
        if on {
            minutes := state.Remaining[mode-1]
            r.RemainingMinutes = &minutes
        }
        return r
    }
    if !changed { return result(before), nil }
    if err := ctx.Err(); err != nil { return timedResult{}, err }
    if err := device.WriteBoostMask(ctx, mask); err != nil {
        return timedResult{}, fmt.Errorf("timed mode command failed; refresh state before retrying: %w", err)
    }
    if attempts < 1 { attempts = 1 }
    var readErr error
    for i := 0; i < attempts; i++ {
        timer := time.NewTimer(interval)
        select {
        case <-ctx.Done():
            timer.Stop()
            return timedResult{}, fmt.Errorf("command sent but timed mode state not verified: %w", ctx.Err())
        case <-timer.C:
        }
        after, err := device.ReadBoost(ctx)
        readErr = err
        if err == nil && timedStateMatches(before, after, mode, on) {
            return result(after), nil
        }
    }
    if readErr != nil {
        return timedResult{}, fmt.Errorf("command acknowledged but timed mode readback failed: %w", readErr)
    }
    return timedResult{}, errors.New("command acknowledged but requested mode/timer not confirmed; refresh state")
}

// BOOST result JSON remains backward-compatible with v0.1.6.
func changeBoost(ctx context.Context, device boostDevice, boost int, on bool, interval time.Duration, attempts int) (boostResult, error) {
    if boost != 1 && boost != 2 { return boostResult{}, errBoostInvalid }
    result, err := changeTimed(ctx, device, boost, on, interval, attempts)
    if err != nil { return boostResult{}, err }
    return boostResult{
        OK: result.OK, Boost: fmt.Sprint(boost), Active: result.Active,
        Changed: result.Changed, Verified: result.Verified, StateMask: result.StateMask,
        RemainingMinutes: result.RemainingMinutes,
    }, nil
}

func changeFireplace(ctx context.Context, device boostDevice, on bool, interval time.Duration, attempts int) (fireplaceResult, error) {
    result, err := changeTimed(ctx, device, 3, on, interval, attempts)
    if err != nil { return fireplaceResult{}, err }
    return fireplaceResult{
        OK: result.OK, Mode: "fireplace", Active: result.Active,
        Changed: result.Changed, Verified: result.Verified, StateMask: result.StateMask,
        RemainingMinutes: result.RemainingMinutes,
    }, nil
}
