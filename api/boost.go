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

func (ws *WebServer) apiVentBoost(w http.ResponseWriter, r *http.Request) {
	boost := chi.URLParam(r, "boost")
	flag := 64
	timer := "u7427"
	if boost == "2" {
		flag = 128
		timer = "u7428"
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
		if err != nil { return nil, err }
		var rows [][]json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil { return nil, err }
		values := make(map[string]float64)
		for _, row := range rows {
			if len(row) < 3 { continue }
			var key string
			if json.Unmarshal(row[0], &key) != nil { continue }
			switch key {
			case "u6639", "u7074", "u7427", "u7428":
				var v float64
				if json.Unmarshal(row[2], &v) == nil { values[key] = v }
			}
		}
		for _, key := range []string{"u6639", "u7074", "u7427", "u7428"} {
			if _, ok := values[key]; !ok { return nil, fmt.Errorf("missing %s", key) }
		}
		return values, nil
	}

	before, err := read()
	if err != nil { http.Error(w, err.Error(), http.StatusBadGateway); return }
	if before["u7074"] != 1 || before["u6639"] != 0 || before["u7427"] != -1 || before["u7428"] != -1 {
		http.Error(w, "ventilation must be on with no active BOOST", http.StatusConflict)
		return
	}

	log.Printf("Starting BOOST %s via u6639=%d", boost, flag)
	resp, err := session.SendInstallationRequest(r.Context(), []econet.OperationRequest{{
		Name: econet.PARAMS_MODIFICATION,
		Targets: []econet.TargetRequest{{
			Component: componentID,
			Parameters: map[string]string{"u6639": fmt.Sprintf("%d", flag)},
		}},
	}})
	if err != nil { http.Error(w, err.Error(), http.StatusBadGateway); return }

	accepted := false
	for _, op := range resp {
		if op.Name != econet.PARAMS_MODIFICATION || op.StatusCode != 0 { continue }
		for _, t := range op.Targets {
			if t.Component == componentID && t.StatusCode == 0 { accepted = true }
		}
	}
	if !accepted { http.Error(w, "BOOST write not acknowledged", http.StatusBadGateway); return }

	time.Sleep(2 * time.Second)
	after, err := read()
	if err != nil { http.Error(w, fmt.Sprintf("BOOST write acknowledged but readback failed: %v", err), http.StatusBadGateway); return }
	if int(after["u6639"])&flag == 0 || after[timer] < 0 {
		http.Error(w, "BOOST write acknowledged but activation not confirmed", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true, "boost": boost, "remaining_minutes": int(after[timer]),
	})
}
