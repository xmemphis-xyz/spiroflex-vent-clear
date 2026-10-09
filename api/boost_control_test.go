package api

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

type fakeBoostDevice struct {
	states []boostSnapshot
	reads int
	writes []uint32
	readErr error
	writeErr error
}
func (f *fakeBoostDevice) ReadBoost(context.Context) (boostSnapshot, error) {
	if f.readErr != nil { return boostSnapshot{}, f.readErr }
	i := f.reads
	f.reads++
	if i >= len(f.states) { i = len(f.states)-1 }
	return f.states[i], nil
}
func (f *fakeBoostDevice) WriteBoostMask(_ context.Context, mask uint32) error {
	f.writes = append(f.writes, mask)
	return f.writeErr
}
func boostState(power bool, mask uint32, first, second int64) boostSnapshot {
	return boostSnapshot{Power: power, Mask: mask, Remaining: [2]int64{first, second}}
}

func TestPlanBoostChange(t *testing.T) {
	cases := []struct {
		name string; state boostSnapshot; boost int; on bool
		mask uint32; changed, conflict bool
	}{
		{"start1", boostState(true,0,-1,-1),1,true,64,true,false},
		{"start2", boostState(true,0,-1,-1),2,true,128,true,false},
		{"stop1", boostState(true,64,13,-1),1,false,0,true,false},
		{"stop2", boostState(true,128,-1,8),2,false,0,true,false},
		{"stop1_preserves_boost2_and_other_bits", boostState(true,64|128|1024,13,8),1,false,128|1024,true,false},
		{"stop2_preserves_boost1_and_other_bits", boostState(true,64|128|1024,13,8),2,false,64|1024,true,false},
		{"stop_allowed_when_power_off", boostState(false,64,13,-1),1,false,0,true,false},
		{"off_is_idempotent", boostState(true,128,-1,8),1,false,128,false,false},
		{"on_does_not_restart_timer", boostState(true,64,12,-1),1,true,64,false,false},
		{"start_rejects_other_boost", boostState(true,128,-1,8),1,true,0,false,true},
		{"start_rejects_other_mode", boostState(true,1024,-1,-1),1,true,0,false,true},
		{"start_rejects_power_off", boostState(false,0,-1,-1),1,true,0,false,true},
		{"inconsistent_timer_not_treated_as_off", boostState(true,0,12,-1),1,false,0,false,true},
	}
	for _, tc := range cases { t.Run(tc.name,func(t *testing.T){
		mask, changed, err := planBoostChange(tc.state,tc.boost,tc.on)
		if (err != nil) != tc.conflict || mask != tc.mask || changed != tc.changed {
			t.Fatalf("got (%d,%t,%v), want (%d,%t,conflict=%t)",mask,changed,err,tc.mask,tc.changed,tc.conflict)
		}
	}) }
}

func TestDecodeBoostSnapshot(t *testing.T) {
	good := []byte(`{"u7074":[1,0,1],"u6639":[64,0,4294967296],"u7427":[13,0,0],"u7428":[-1,0,0]}`)
	state, err := decodeBoostSnapshot(good)
	if err != nil || state != boostState(true,64,13,-1) { t.Fatalf("%+v %v",state,err) }
	for _, value := range []string{"null", "64.5", "-1", "4294967296", `"64"`, "true", "{}"} {
		raw := []byte(fmt.Sprintf(`{"u7074":[1],"u6639":[%s],"u7427":[13],"u7428":[-1]}`,value))
		if _,err := decodeBoostSnapshot(raw); err == nil { t.Fatalf("accepted mask %s",value) }
	}
	for _, raw := range []string{`null`,`{}`,`[]`,`{"u7074":[]}`,`{"u7074":[null]}`} {
		if _,err := decodeBoostSnapshot([]byte(raw)); err == nil { t.Fatalf("accepted %s",raw) }
	}
}

func TestChangeBoostStopVerification(t *testing.T) {
	on := boostState(true,64,13,-1)
	off := boostState(true,0,-1,-1)
	device := &fakeBoostDevice{states: []boostSnapshot{on,on,off}}
	result,err := changeBoost(context.Background(),device,1,false,0,3)
	if err != nil || !result.Verified || result.Active || result.RemainingMinutes != nil || !result.Changed { t.Fatalf("%+v %v",result,err) }
	if len(device.writes)!=1 || device.writes[0]!=0 { t.Fatalf("writes=%v",device.writes) }
}
func TestChangeBoostAcknowledgementAloneNotEnough(t *testing.T) {
	device := &fakeBoostDevice{states: []boostSnapshot{boostState(true,64,13,-1)}}
	if _,err := changeBoost(context.Background(),device,1,false,0,3); err == nil { t.Fatal("unverified STOP accepted") }
	if len(device.writes)!=1 { t.Fatal("write was retried") }
}
func TestChangeBoostDoesNotAcceptUnrelatedTimer(t *testing.T) {
	device := &fakeBoostDevice{states: []boostSnapshot{boostState(true,0,-1,-1),boostState(true,64,-1,10)}}
	if _,err := changeBoost(context.Background(),device,1,true,0,1); err == nil { t.Fatal("wrong BOOST timer accepted") }
}
func TestStopMustConfirmTimerAndPower(t *testing.T) {
	before := boostState(true,64,12,-1)
	for _, after := range []boostSnapshot{boostState(true,0,12,-1),boostState(false,0,-1,-1),boostState(true,128,-1,10)} {
		if boostStateMatches(before,after,1,false) { t.Fatalf("accepted %+v",after) }
	}
}
func TestChangeBoostNoWriteWhenInactive(t *testing.T) {
	device := &fakeBoostDevice{states: []boostSnapshot{boostState(true,128,-1,10)}}
	result,err := changeBoost(context.Background(),device,1,false,0,1)
	if err != nil || result.Changed || len(device.writes)!=0 { t.Fatalf("%+v %v writes=%v",result,err,device.writes) }
}
func TestChangeBoostReadFailurePreventsWrite(t *testing.T) {
	device := &fakeBoostDevice{readErr: errors.New("offline")}
	if _,err := changeBoost(context.Background(),device,1,false,0,1); err == nil || len(device.writes)!=0 { t.Fatal("read failure allowed a write") }
}
func TestChangeBoostWriteErrorDoesNotRetry(t *testing.T) {
	device := &fakeBoostDevice{states: []boostSnapshot{boostState(true,64,12,-1)},writeErr: errors.New("rejected")}
	if _,err := changeBoost(context.Background(),device,1,false,0,3); err == nil || len(device.writes)!=1 { t.Fatal("write error ignored or retried") }
}
func TestChangeBoostCancelledAndInvalid(t *testing.T) {
	device := &fakeBoostDevice{}
	ctx,cancel := context.WithCancel(context.Background()); cancel()
	if _,err := changeBoost(ctx,device,1,false,0,1); !errors.Is(err,context.Canceled) { t.Fatal(err) }
	if _,err := changeBoost(context.Background(),device,3,false,0,1); !errors.Is(err,errBoostInvalid) { t.Fatal(err) }
	if device.reads!=0 || len(device.writes)!=0 { t.Fatal("unexpected device access") }
}
