package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mtojek/spiroflex-vent-clear/econet"
)

type mqttBoostDevice struct {
	session *econet.MQTTSession
	componentID string
}

func (d mqttBoostDevice) ReadBoost(ctx context.Context) (boostSnapshot, error) {
	values, err := d.session.GetValues(ctx, d.componentID, []string{"u7074", "u6639", "u7427", "u7428"})
	if err != nil { return boostSnapshot{}, err }
	for _, target := range values {
		if target.Component != d.componentID { continue }
		if target.StatusCode != 0 { return boostSnapshot{}, fmt.Errorf("BOOST read failed: status %d", target.StatusCode) }
		return decodeBoostSnapshot(target.Parameters)
	}
	return boostSnapshot{}, errors.New("BOOST component missing from response")
}

func (d mqttBoostDevice) WriteBoostMask(ctx context.Context, mask uint32) error {
	log.Printf("BOOST command: u6639=%d", mask)
	resp, err := d.session.SendInstallationRequest(ctx, []econet.OperationRequest{{
		Name: econet.PARAMS_MODIFICATION,
		Targets: []econet.TargetRequest{{Component: d.componentID, Parameters: map[string]string{"u6639": strconv.FormatUint(uint64(mask), 10)}}},
	}})
	if err != nil { return err }
	accepted := false
	for _, op := range resp {
		if op.Name != econet.PARAMS_MODIFICATION { continue }
		if op.StatusCode != 0 { return fmt.Errorf("BOOST command rejected: status %d", op.StatusCode) }
		for _, target := range op.Targets {
			if target.Component != d.componentID { continue }
			if target.StatusCode != 0 { return fmt.Errorf("BOOST target rejected: status %d", target.StatusCode) }
			accepted = true
		}
	}
	if !accepted { return errors.New("BOOST command not acknowledged") }
	return nil
}

func (ws *WebServer) apiVentBoost(w http.ResponseWriter, r *http.Request) {
	ws.serveBoostChange(w, r, true)
}

func (ws *WebServer) apiVentBoostOff(w http.ResponseWriter, r *http.Request) {
	ws.serveBoostChange(w, r, false)
}

func writeBoostError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response{Error: err.Error()})
}

func (ws *WebServer) serveBoostChange(w http.ResponseWriter, r *http.Request, on bool) {
	boost, err := strconv.Atoi(chi.URLParam(r, "boost"))
	if err != nil || (boost != 1 && boost != 2) { writeBoostError(w, errBoostInvalid, http.StatusBadRequest); return }
	// Covers start/stop and the legacy diagnostic write within this server.
	if !ws.boostMu.TryLock() {
		writeBoostError(w, errors.New("another BOOST command is in progress"), http.StatusConflict)
		return
	}
	defer ws.boostMu.Unlock()
	ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
	defer cancel()
	session, componentID, err := ws.prepareEconet(ctx)
	if err != nil { writeBoostError(w, err, http.StatusBadGateway); return }
	defer session.Disconnect()
	result, err := changeBoost(ctx, mqttBoostDevice{session, componentID}, boost, on, time.Second, 5)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errBoostConflict) { status = http.StatusConflict }
		if errors.Is(err, context.DeadlineExceeded) { status = http.StatusGatewayTimeout }
		log.Printf("BOOST %d on=%t failed: %v", boost, on, err)
		writeBoostError(w, err, status)
		return
	}
	log.Printf("BOOST %d on=%t verified (changed=%t)", boost, on, result.Changed)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(result)
}
