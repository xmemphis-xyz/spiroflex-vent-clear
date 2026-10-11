package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mtojek/spiroflex-vent-clear/econet"
)

// apiVentFireplaceDiagnostics collects read-only observations for mapping
// the third timed mode (Kominek). Do not infer a writable bit from the
// BOOST bits without comparing OFF and ON readings on the actual controller.
func (ws *WebServer) apiVentFireplaceDiagnostics(w http.ResponseWriter, r *http.Request) {
	session, componentID, err := ws.prepareEconet(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer session.Disconnect()

	// u6639 is the timed-mode status mask (BOOST 1/2 already mapped).
	// u7429 and u7158 are candidates for the third timer and duration.
	// Include fan output to verify what the mode actually does.
	params := []string{
		"u7074", "u6639",
		"u7427", "u7428", "u7429",
		"u7099", "u7100", "u7158",
		"u6202", "u6203", "u7151",
		"u7152", "u7153", "u7159", "u7160",
	}
	values, err := session.GetValues(r.Context(), componentID, params)
	if err != nil {
		http.Error(w, fmt.Sprintf("fireplace diagnostic read failed: %v", err), http.StatusBadGateway)
		return
	}
	parameters, err := findVentParameters(values, componentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	observations := make(map[string]any, len(params))
	for _, name := range params {
		if values, ok := parameters[name]; ok && len(values) > 0 {
			observations[name] = values[0]
		} else {
			observations[name] = nil
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"read_only":              true,
		"fireplace_command_mapped": false,
		"parameters":             observations,
	})
}
