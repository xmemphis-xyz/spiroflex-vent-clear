package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// Retain the diagnostic URL, but use the same lock and validation as normal ON.
func (ws *WebServer) apiVentBoostTest(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("confirm") == "experimental-write" {
		ws.serveBoostChange(w, r, true)
		return
	}
	boost, err := strconv.Atoi(chi.URLParam(r, "boost"))
	if err != nil || (boost != 1 && boost != 2) { writeBoostError(w, errBoostInvalid, http.StatusBadRequest); return }
	ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
	defer cancel()
	session, componentID, err := ws.prepareEconet(ctx)
	if err != nil { writeBoostError(w, err, http.StatusBadGateway); return }
	defer session.Disconnect()
	before, err := (mqttBoostDevice{session, componentID}).ReadBoost(ctx)
	if err != nil { writeBoostError(w, err, http.StatusBadGateway); return }
	mask, changed, err := planBoostChange(before, boost, true)
	if err != nil { writeBoostError(w, err, http.StatusConflict); return }
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"dry_run": true, "boost": strconv.Itoa(boost), "candidate_register": "u6639",
		"candidate_value": mask, "write_needed": changed,
		"before": map[string]any{"u6639": before.Mask, "power": before.Power, "u7427": before.Remaining[0], "u7428": before.Remaining[1]},
	})
}
