package api

import (
	"encoding/json"
	"fmt"
	"net/http"

)

type boostParameter struct {
	Value *float64 `json:"value"`
	Type  any      `json:"type,omitempty"`
	Min   any      `json:"min,omitempty"`
	Max   any      `json:"max,omitempty"`
}

func (ws *WebServer) apiVentBoostDiagnostics(w http.ResponseWriter, r *http.Request) {
	session, componentID, err := ws.prepareEconet(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer session.Disconnect()

	raw, err := session.GetParameterTable(r.Context(), componentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	var rows [][]json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		http.Error(w, fmt.Sprintf("invalid parameter table: %v", err), http.StatusBadGateway)
		return
	}

	keys := map[string]bool{
		"u6639": true, "u6635": true, "u6636": true,
		"u6637": true, "u6638": true, "u7099": true,
		"u7100": true, "u7427": true, "u7428": true,
	}
	parameters := make(map[string]boostParameter)
	for _, row := range rows {
		if len(row) < 3 || string(row[0]) == "null" {
			continue
		}
		var key string
		if err := json.Unmarshal(row[0], &key); err != nil || !keys[key] {
			continue
		}
		var value *float64
		if string(row[2]) != "null" {
			var number float64
			if err := json.Unmarshal(row[2], &number); err != nil {
				continue
			}
			value = &number
		}
		item := boostParameter{Value: value}
		if len(row) > 1 {
			_ = json.Unmarshal(row[1], &item.Type)
		}
		if len(row) > 3 {
			_ = json.Unmarshal(row[3], &item.Min)
		}
		if len(row) > 4 {
			_ = json.Unmarshal(row[4], &item.Max)
		}
		parameters[key] = item
	}

	mask := 0
	if flag, ok := parameters["u6639"]; ok && flag.Value != nil {
		mask = int(*flag.Value)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"boost_1_active": mask&64 != 0,
		"boost_2_active": mask&128 != 0,
		"state_mask": mask,
		"parameters": parameters,
		"read_only": true,
	})
}
