package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mtojek/spiroflex-vent-clear/econet"
)

func (ws *WebServer) apiVentBoostTest(w http.ResponseWriter, r *http.Request) {
	boost := chi.URLParam(r, "boost")
	flag := 64
	if boost == "2" {
		flag = 128
	} else if boost != "1" {
		http.Error(w, "BOOST must be 1 or 2", http.StatusBadRequest)
		return
	}

	session, componentID, err := ws.prepareEconet(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer session.Disconnect()

	read := func() (map[string]float64, error) {
		raw, err := session.GetParameterTable(r.Context(), componentID)
		if err != nil {
			return nil, err
		}
		var rows [][]json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, err
		}
		values := make(map[string]float64)
		for _, row := range rows {
			if len(row) < 3 {
				continue
			}
			var key string
			if json.Unmarshal(row[0], &key) != nil {
				continue
			}
			if key != "u6639" && key != "u7074" && key != "u7427" && key != "u7428" {
				continue
			}
			var value float64
			if json.Unmarshal(row[2], &value) == nil {
				values[key] = value
			}
		}
		for _, key := range []string{"u6639", "u7074", "u7427", "u7428"} {
			if _, ok := values[key]; !ok {
				return nil, fmt.Errorf("missing parameter %s", key)
			}
		}
		return values, nil
	}

	before, err := read()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if before["u7074"] != 1 || before["u6639"] != 0 ||
		before["u7427"] != -1 || before["u7428"] != -1 {
		http.Error(w, "test requires powered-on ventilation and no active BOOST", http.StatusConflict)
		return
	}

	result := map[string]any{
		"boost": boost, "candidate_register": "u6639",
		"candidate_value": flag, "before": before,
		"warning": "Experimental write; u6639 may be read-only. No automatic rollback.",
	}
	if r.URL.Query().Get("confirm") != "experimental-write" {
		result["dry_run"] = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
		return
	}

	log.Printf("BOOST diagnostic: experimental u6639=%d requested", flag)
	resp, err := session.SendInstallationRequest(r.Context(), []econet.OperationRequest{{
		Name: econet.PARAMS_MODIFICATION,
		Targets: []econet.TargetRequest{{
			Component: componentID,
			Parameters: map[string]string{"u6639": fmt.Sprintf("%d", flag)},
		}},
	}})
	if err != nil {
		http.Error(w, fmt.Sprintf("experimental write failed: %v", err), http.StatusBadGateway)
		return
	}
	accepted := false
	for _, op := range resp {
		if op.Name != econet.PARAMS_MODIFICATION || op.StatusCode != 0 {
			continue
		}
		for _, target := range op.Targets {
			if target.Component == componentID && target.StatusCode == 0 {
				accepted = true
			}
		}
	}
	result["acknowledged"] = accepted
	if !accepted {
		result["error"] = "write not acknowledged"
		w.WriteHeader(http.StatusBadGateway)
	} else {
		time.Sleep(2 * time.Second)
		after, readErr := read()
		if readErr != nil {
			result["readback_error"] = readErr.Error()
		} else {
			result["after"] = after
			result["activated"] = int(after["u6639"])&flag != 0 &&
				(after["u7427"] >= 0 || after["u7428"] >= 0)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
