package api

import (
    "context"
    "encoding/json"
    "errors"
    "log"
    "net/http"
    "time"
)

func (ws *WebServer) apiVentFireplaceOn(w http.ResponseWriter, r *http.Request) {
    ws.serveFireplaceChange(w, r, true)
}

func (ws *WebServer) apiVentFireplaceOff(w http.ResponseWriter, r *http.Request) {
    ws.serveFireplaceChange(w, r, false)
}

func (ws *WebServer) serveFireplaceChange(w http.ResponseWriter, r *http.Request, on bool) {
    if !ws.boostMu.TryLock() {
        writeBoostError(w, errors.New("another timed mode command is in progress"), http.StatusConflict)
        return
    }
    defer ws.boostMu.Unlock()

    ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
    defer cancel()
    session, componentID, err := ws.prepareEconet(ctx)
    if err != nil {
        writeBoostError(w, err, http.StatusBadGateway)
        return
    }
    defer session.Disconnect()

    result, err := changeFireplace(ctx, mqttBoostDevice{session, componentID}, on, time.Second, 5)
    if err != nil {
        status := http.StatusBadGateway
        if errors.Is(err, errBoostConflict) { status = http.StatusConflict }
        if errors.Is(err, context.DeadlineExceeded) { status = http.StatusGatewayTimeout }
        log.Printf("Fireplace on=%t failed: %v", on, err)
        writeBoostError(w, err, status)
        return
    }
    log.Printf("Fireplace on=%t verified (changed=%t)", on, result.Changed)
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "no-store")
    _ = json.NewEncoder(w).Encode(result)
}
