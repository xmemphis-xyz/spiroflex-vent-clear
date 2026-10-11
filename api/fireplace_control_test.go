package api

import (
    "context"
    "errors"
    "fmt"
    "testing"
)

func timedState(power bool, mask uint32, first, second, third int64) boostSnapshot {
    return boostSnapshot{Power: power, Mask: mask, Remaining: [3]int64{first, second, third}}
}

func TestPlanFireplaceChange(t *testing.T) {
    cases := []struct {
        name string
        state boostSnapshot
        on bool
        wantMask uint32
        wantWrite bool
        conflict bool
    }{
        {"start", timedState(true,0,-1,-1,-1),true,256,true,false},
        {"stop", timedState(true,256,-1,-1,10),false,0,true,false},
        {"stop_preserves_other_mode", timedState(true,64|256|1024,5,-1,10),false,64|1024,true,false},
        {"stop_when_power_off", timedState(false,256,-1,-1,10),false,0,true,false},
        {"stop_inactive_idempotent", timedState(true,64,5,-1,-1),false,64,false,false},
        {"start_does_not_rearm", timedState(true,256,-1,-1,7),true,256,false,false},
        {"start_rejects_boost", timedState(true,128,-1,4,-1),true,0,false,true},
        {"start_rejects_unknown_mode", timedState(true,1024,-1,-1,-1),true,0,false,true},
        {"start_rejects_power_off", timedState(false,0,-1,-1,-1),true,0,false,true},
        {"start_rejects_active_timer_without_bit", timedState(true,0,-1,-1,7),true,0,false,true},
        {"stop_rejects_timer_without_bit", timedState(true,0,-1,-1,7),false,0,false,true},
        {"start_rejects_overlap", timedState(true,64|256,5,-1,8),true,0,false,true},
        {"start_rejects_bad_other_timer", timedState(true,256,4,-1,8),true,0,false,true},
    }
    for _, tc := range cases {
        t.Run(tc.name,func(t *testing.T){
            mask, changed, err := planFireplaceChange(tc.state,tc.on)
            if (err!=nil)!=tc.conflict || mask!=tc.wantMask || changed!=tc.wantWrite {
                t.Fatalf("got mask=%d changed=%t err=%v want mask=%d changed=%t conflict=%t",mask,changed,err,tc.wantMask,tc.wantWrite,tc.conflict)
            }
        })
    }
}

func TestBoostCannotStartWhileFireplaceRunning(t *testing.T) {
    state:=timedState(true,256,-1,-1,10)
    for _, boost:=range []int{1,2} {
        if _,changed,err:=planBoostChange(state,boost,true); !errors.Is(err,errBoostConflict)||changed {
            t.Fatalf("boost=%d changed=%t error=%v",boost,changed,err)
        }
        if mask,changed,err:=planBoostChange(state,boost,false);err!=nil||changed||mask!=256 {
            t.Fatalf("off boosted third mode: %d %t %v",mask,changed,err)
        }
    }
    if _,changed,err:=planBoostChange(timedState(true,0,-1,-1,10),1,true);!errors.Is(err,errBoostConflict)||changed {
        t.Fatalf("boost started with stale fireplace countdown: %t %v",changed,err)
    }
}

func TestDecodeFireplaceSnapshot(t *testing.T) {
    good:=[]byte(`{"u7074":[1],"u6639":[256],"u7427":[-1],"u7428":[-1],"u7429":[10]}`)
    state,err:=decodeBoostSnapshot(good)
    if err!=nil||state!=timedState(true,256,-1,-1,10) {t.Fatalf("%+v %v",state,err)}
    for _,bad:=range []string{`null`,`-2`,`0.5`,`"10"`,`4294967296`} {
        raw:=[]byte(fmt.Sprintf(`{"u7074":[1],"u6639":[256],"u7427":[-1],"u7428":[-1],"u7429":[%s]}`,bad))
        if _,err:=decodeBoostSnapshot(raw);err==nil {t.Fatalf("accepted bad fireplace timer %s",bad)}
    }
    if _,err:=decodeBoostSnapshot([]byte(`{"u7074":[1],"u6639":[256],"u7427":[-1],"u7428":[-1]}`));err==nil {
        t.Fatal("missing fireplace timer accepted")
    }
}

func TestFireplaceStartAndStopConfirmed(t *testing.T) {
    idle:=timedState(true,0,-1,-1,-1)
    active:=timedState(true,256,-1,-1,10)
    start:=&fakeBoostDevice{states:[]boostSnapshot{idle,idle,active}}
    on,err:=changeFireplace(context.Background(),start,true,0,3)
    if err!=nil||!on.Verified||!on.Active||on.RemainingMinutes==nil||*on.RemainingMinutes!=10||on.Mode!="fireplace" {
        t.Fatalf("ON: %+v %v",on,err)
    }
    if len(start.writes)!=1||start.writes[0]!=256 {t.Fatalf("ON writes: %v",start.writes)}

    stop:=&fakeBoostDevice{states:[]boostSnapshot{active,active,idle}}
    off,err:=changeFireplace(context.Background(),stop,false,0,3)
    if err!=nil||!off.Verified||off.Active||off.RemainingMinutes!=nil||off.Mode!="fireplace" {
        t.Fatalf("OFF: %+v %v",off,err)
    }
    if len(stop.writes)!=1||stop.writes[0]!=0 {t.Fatalf("OFF writes: %v",stop.writes)}
}

func TestFireplaceReadbackAndUnexpectedPowerAreRejected(t *testing.T) {
    active:=timedState(true,256,-1,-1,10)
    wrong:=[]boostSnapshot{
        timedState(true,0,-1,-1,10), // cleared bit but timer remains
        timedState(false,0,-1,-1,-1), // ventilation power changed
        timedState(true,64,-1,-1,-1), // another bit was changed
    }
    for _,after:=range wrong {
        if timedStateMatches(active,after,3,false){t.Fatalf("accepted %+v",after)}
    }
    device:=&fakeBoostDevice{states:[]boostSnapshot{active,active,active}}
    if _,err:=changeFireplace(context.Background(),device,false,0,2);err==nil {
        t.Fatal("accepted unverified fireplace stop")
    }
    if len(device.writes)!=1 {t.Fatalf("write retried: %v",device.writes)}
}

func TestFireplaceCommandRejectedWhenAlreadyBoosting(t *testing.T) {
    device:=&fakeBoostDevice{states:[]boostSnapshot{timedState(true,64,7,-1,-1)}}
    if _,err:=changeFireplace(context.Background(),device,true,0,1);!errors.Is(err,errBoostConflict) {
        t.Fatalf("expected conflict, got %v",err)
    }
    if len(device.writes)!=0 {t.Fatalf("unsafe write: %v",device.writes)}
}

func TestFireplaceNoWriteWhenAlreadyActive(t *testing.T) {
    device:=&fakeBoostDevice{states:[]boostSnapshot{timedState(true,256,-1,-1,7)}}
    result,err:=changeFireplace(context.Background(),device,true,0,1)
    if err!=nil||result.Changed||len(device.writes)!=0||result.RemainingMinutes==nil||*result.RemainingMinutes!=7 {
        t.Fatalf("%+v %v writes=%v",result,err,device.writes)
    }
}
