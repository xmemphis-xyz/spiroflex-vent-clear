package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mtojek/spiroflex-vent-clear/econet"
	"log"
	"net/http"
	"strconv"
)

func (ws *WebServer) ventLevel(ctx context.Context, levelStr string) error {
	level, err := strconv.Atoi(levelStr)
	if err != nil {
		return fmt.Errorf("can't convert level to int: %w", err)
	}

	if level < 1 || level > 3 {
		return fmt.Errorf("level must be between 1 and 3")
	}

	level += 2 // levels are enumerated from 3 to 5

	session, targetComponentID, err := ws.prepareEconet(ctx)
	if err != nil {
		return err
	}

	err = session.VentLevel(ctx, targetComponentID, fmt.Sprintf("%d", level))
	if err != nil {
		return fmt.Errorf("unable to modify parameters: %w", err)
	}
	return nil
}

func (ws *WebServer) ventPause(ctx context.Context) error {
	session, targetComponentID, err := ws.prepareEconet(ctx)
	if err != nil {
		return err
	}

	err = session.VentPause(ctx, targetComponentID)
	if err != nil {
		return fmt.Errorf("unable to modify parameters: %w", err)
	}
	return nil
}

func (ws *WebServer) ventMode(ctx context.Context, mode string) error {
	if mode == "schedule" {
		mode = econet.PARAM_MODE_SCHEDULE
	} else {
		mode = econet.PARAM_MODE_MANUAL
	}

	session, targetComponentID, err := ws.prepareEconet(ctx)
	if err != nil {
		return err
	}

	err = session.VentMode(ctx, targetComponentID, mode)
	if err != nil {
		return fmt.Errorf("unable to modify parameters: %w", err)
	}
	return nil
}

func (ws *WebServer) ventPower(ctx context.Context, state string) error {
	if state == "on" {
		state = econet.PARAM_POWER_ON
	} else {
		state = econet.PARAM_POWER_OFF
	}

	session, targetComponentID, err := ws.prepareEconet(ctx)
	if err != nil {
		return err
	}

	err = session.VentPower(ctx, targetComponentID, state)
	if err != nil {
		return fmt.Errorf("unable to modify parameters: %w", err)
	}
	return nil
}

func (ws *WebServer) prepareEconet(ctx context.Context) (*econet.MQTTSession, string, error) {
	client, err := econet.New(ctx, ws.c)
	if err != nil {
		return nil, "", fmt.Errorf("unable to create client: %w", err)
	}

	if ws.c.Installation.ID == "" {
		return nil, "", fmt.Errorf("installation ID is not configured")
	}

	session, err := client.MQTT(ctx, ws.c.Installation.ID)
	if err != nil {
		return nil, "", fmt.Errorf("MQTT error: %w", err)
	}

	gcob, err := session.GetComponentsOnBus(ctx)
	if err != nil {
		session.Disconnect()
		return nil, "", fmt.Errorf("unable to fetch components on bus: %w", err)
	}

	var targetComponentID string

	for _, c := range gcob {
		if c.ComponentName == "ecoVENT Simple" {
			targetComponentID = c.ComponentID
			break
		}
	}

	if targetComponentID == "" {
		session.Disconnect()
		return nil, "", fmt.Errorf("ecoVENT Simple component not found")
	}

	return session, targetComponentID, nil
}

func (ws *WebServer) apiVentDiagnostics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("DIAG: starting Cognito authentication")

	client, err := econet.New(ctx, ws.c)
	if err != nil {
		log.Printf("DIAG: Cognito error: %v", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	log.Printf("DIAG: Cognito authentication OK")

	if ws.c.Installation.ID == "" {
		log.Printf("DIAG: installation ID is empty")
		http.Error(w, "installation ID is not configured", http.StatusInternalServerError)
		return
	}

	log.Printf("DIAG: connecting MQTT to installation %s", ws.c.Installation.ID)

	session, err := client.MQTT(ctx, ws.c.Installation.ID)
	if err != nil {
		log.Printf("DIAG: MQTT error: %v", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	log.Printf("DIAG: MQTT connection OK")

	defer session.Disconnect()

	log.Printf("DIAG: requesting GET_COMPONENTS_ON_BUS")

	components, err := session.GetComponentsOnBus(ctx)
	if err != nil {
		log.Printf("DIAG: GET_COMPONENTS_ON_BUS error: %v", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	log.Printf("DIAG: GET_COMPONENTS_ON_BUS OK, components=%d", len(components))

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"installation": map[string]string{
			"id":   ws.c.Installation.ID,
			"name": ws.c.Installation.Name,
		},
		"components": components,
	}); err != nil {
		log.Printf("DIAG: JSON error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) apiVentValues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := econet.New(ctx, ws.c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if ws.c.Installation.ID == "" {
		http.Error(w, "installation ID is not configured", http.StatusInternalServerError)
		return
	}

	session, err := client.MQTT(ctx, ws.c.Installation.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer session.Disconnect()

	components, err := session.GetComponentsOnBus(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	var componentID string

	for _, component := range components {
		if component.ComponentName == "ecoVENT Simple" {
			componentID = component.ComponentID
			break
		}
	}

	if componentID == "" {
		http.Error(w, "ecoVENT Simple component not found", http.StatusNotFound)
		return
	}

	values, err := session.GetValues(ctx, componentID, []string{
		"u7074",
		"u81",
		"u6630",
		"u6205",
		"u6207",
		"u6209",
		"u6208",
		"u6338",
		"u6273",
		"u6265",
		"u7151",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(values); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) apiVentSequentialValues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := econet.New(ctx, ws.c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if ws.c.Installation.ID == "" {
		http.Error(w, "installation ID is not configured", http.StatusInternalServerError)
		return
	}

	session, err := client.MQTT(ctx, ws.c.Installation.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer session.Disconnect()

	components, err := session.GetComponentsOnBus(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	var componentID string

	for _, component := range components {
		if component.ComponentName == "ecoVENT Simple" {
			componentID = component.ComponentID
			break
		}
	}

	if componentID == "" {
		http.Error(w, "ecoVENT Simple component not found", http.StatusNotFound)
		return
	}

	values, err := session.GetSequentialValues(ctx, componentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(values)
}

func (ws *WebServer) apiVentParameterTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	session, componentID, err := ws.prepareEconet(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer session.Disconnect()

	values, err := session.GetParameterTable(ctx, componentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(values)
}
