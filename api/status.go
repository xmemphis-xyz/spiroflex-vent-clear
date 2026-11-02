package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/mtojek/spiroflex-vent-clear/econet"
)

type VentStatus struct {
	Power               bool    `json:"power"`
	Level               *int    `json:"level"`
	Paused              bool    `json:"paused"`
	Mode                string  `json:"mode"`
	Status              string  `json:"status"`
	SupplyTemperature   float64 `json:"supply_temperature"`
	ExtractTemperature  float64 `json:"extract_temperature"`
	IntakeTemperature   float64 `json:"intake_temperature"`
	ExhaustTemperature  float64 `json:"exhaust_temperature"`
	RoomTemperature     float64 `json:"room_temperature"`
	Humidity            float64 `json:"humidity"`
	SupplyFanPercent    float64 `json:"supply_fan_percent"`
	ExtractFanPercent   float64 `json:"extract_fan_percent"`
	FilterDaysRemaining float64 `json:"filter_days_remaining"`
	SupplyFilterUsage   float64 `json:"supply_filter_usage"`
	ExtractFilterUsage  float64 `json:"extract_filter_usage"`
	Alarm               bool    `json:"alarm"`
	Boost1Active        bool    `json:"boost_1_active"`
	Boost2Active        bool    `json:"boost_2_active"`
	FireplaceActive     bool    `json:"fireplace_active"`
	Boost1Remaining     *int    `json:"boost_1_remaining"`
	Boost2Remaining     *int    `json:"boost_2_remaining"`
	FireplaceRemaining  *int    `json:"fireplace_remaining"`
}

func (ws *WebServer) apiVentStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	session, componentID, err := ws.prepareEconet(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer session.Disconnect()

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
		"u7151",
		"u6202",
		"u6203",
		"u7076",
		"u6938",
		"u6939",
		"u6999",
		"u6639",
		"u7427",
		"u7428",
		"u7429",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	parameters, err := findVentParameters(values, componentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	status, err := buildVentStatus(parameters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func findVentParameters(values []econet.TargetResponse, componentID string) (map[string][]float64, error) {
	for _, value := range values {
		if value.Component != componentID {
			continue
		}

		parameters, failed, err := decodeParameterValues(value.Parameters)
		if err != nil {
			if value.StatusCode != 0 {
				return nil, fmt.Errorf("GET_VALUES target failed, status code: %d", value.StatusCode)
			}
			return nil, err
		}

		// A non-zero target status with a parameter map means a partial
		// read: some parameters (e.g. u7151 on some controllers) return
		// "err" while the rest are valid. Keep the valid ones.
		if len(failed) > 0 || value.StatusCode != 0 {
			log.Printf("GET_VALUES partial read (status code %d), unreadable parameters: %v", value.StatusCode, failed)
		}
		if len(parameters) == 0 {
			return nil, fmt.Errorf("GET_VALUES target failed, status code: %d", value.StatusCode)
		}

		return parameters, nil
	}

	return nil, fmt.Errorf("component %s not found in GET_VALUES response", componentID)
}

// decodeParameterValues decodes a GET_VALUES parameter map. Each value is
// normally a list with one number; a controller may instead return "err" (as a
// string or inside the list) for a parameter it cannot read. Such parameters
// are left out of the result and reported in failed.
func decodeParameterValues(raw json.RawMessage) (map[string][]float64, []string, error) {
	var rawParameters map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawParameters); err != nil {
		return nil, nil, fmt.Errorf("unable to decode GET_VALUES parameters: %w", err)
	}

	parameters := make(map[string][]float64, len(rawParameters))
	var failed []string
	for name, value := range rawParameters {
		var list []float64
		if err := json.Unmarshal(value, &list); err == nil {
			parameters[name] = list
			continue
		}
		var single float64
		if err := json.Unmarshal(value, &single); err == nil {
			parameters[name] = []float64{single}
			continue
		}
		failed = append(failed, name)
	}
	sort.Strings(failed)
	return parameters, failed, nil
}

func buildVentStatus(parameters map[string][]float64) (VentStatus, error) {
	power, err := requiredParameter(parameters, "u7074")
	if err != nil {
		return VentStatus{}, err
	}

	levelValue, err := requiredParameter(parameters, "u81")
	if err != nil {
		return VentStatus{}, err
	}

	modeValue, err := requiredParameter(parameters, "u6630")
	if err != nil {
		return VentStatus{}, err
	}

	status := VentStatus{
		Power:  power != 0,
		Mode:   modeName(int(modeValue)),
		Status: "unknown",
	}

	// u7151 (operating status) is informational and returns "err" on some
	// controllers, so it must not fail the whole status read.
	if statusValue, err := requiredParameter(parameters, "u7151"); err == nil {
		status.Status = statusName(int(statusValue))
	}

	switch int(levelValue) {
	case 3, 4, 5:
		level := int(levelValue) - 2
		status.Level = &level
	case 6:
		status.Paused = true
	}

	if !status.Power {
		status.Status = "off"
	}

	status.SupplyTemperature, err = requiredParameter(parameters, "u6205")
	if err != nil {
		return VentStatus{}, err
	}

	status.ExtractTemperature, err = requiredParameter(parameters, "u6207")
	if err != nil {
		return VentStatus{}, err
	}

	status.IntakeTemperature, err = requiredParameter(parameters, "u6209")
	if err != nil {
		return VentStatus{}, err
	}

	status.ExhaustTemperature, err = requiredParameter(parameters, "u6208")
	if err != nil {
		return VentStatus{}, err
	}

	status.RoomTemperature, err = requiredParameter(parameters, "u6338")
	if err != nil {
		return VentStatus{}, err
	}

	status.Humidity, err = requiredParameter(parameters, "u6273")
	if err != nil {
		return VentStatus{}, err
	}

	status.SupplyFanPercent, err = requiredParameter(parameters, "u6202")
	if err != nil {
		return VentStatus{}, err
	}

	status.ExtractFanPercent, err = requiredParameter(parameters, "u6203")
	if err != nil {
		return VentStatus{}, err
	}

	status.FilterDaysRemaining, err = requiredParameter(parameters, "u7076")
	if err != nil {
		return VentStatus{}, err
	}

	status.SupplyFilterUsage, err = requiredParameter(parameters, "u6938")
	if err != nil {
		return VentStatus{}, err
	}

	status.ExtractFilterUsage, err = requiredParameter(parameters, "u6939")
	if err != nil {
		return VentStatus{}, err
	}

	alarm, err := requiredParameter(parameters, "u6999")
	if err != nil {
		return VentStatus{}, err
	}
	status.Alarm = alarm != 0


	mask, err := requiredParameter(parameters, "u6639")
	if err != nil { return VentStatus{}, err }
	status.Boost1Active = int(mask)&64 != 0
	status.Boost2Active = int(mask)&128 != 0
	status.FireplaceActive = int(mask)&256 != 0
	remaining1, err := requiredParameter(parameters, "u7427")
	if err != nil { return VentStatus{}, err }
	remaining2, err := requiredParameter(parameters, "u7428")
	if err != nil { return VentStatus{}, err }
	if status.Boost1Active && remaining1 >= 0 {
		minutes := int(remaining1)
		status.Boost1Remaining = &minutes
	}
	if status.Boost2Active && remaining2 >= 0 {
		minutes := int(remaining2)
		status.Boost2Remaining = &minutes
	}
	remaining3, err := requiredParameter(parameters, "u7429")
	if err != nil { return VentStatus{}, err }
	if status.FireplaceActive && remaining3 >= 0 {
		minutes := int(remaining3)
		status.FireplaceRemaining = &minutes
	}
	return status, nil
}

func requiredParameter(parameters map[string][]float64, name string) (float64, error) {
	values, ok := parameters[name]
	if !ok || len(values) == 0 {
		return 0, fmt.Errorf("parameter %s missing from GET_VALUES response", name)
	}
	return values[0], nil
}

func modeName(value int) string {
	switch value {
	case 0:
		return "manual"
	case 1:
		return "schedule"
	default:
		return "unknown"
	}
}

func statusName(value int) string {
	switch value {
	case 0:
		return "none"
	case 1:
		return "off"
	case 2:
		return "stopped"
	case 3:
		return "normal"
	case 4:
		return "heating"
	case 5:
		return "cooling"
	case 6:
		return "dehumidifying"
	case 7:
		return "heat_exchanger_cleaning"
	case 8:
		return "alarm_ventilation"
	case 9:
		return "heater_cooling"
	case 10:
		return "filter_test"
	case 11:
		return "boost"
	case 12:
		return "9600"
	case 13:
		return "start_delay"
	case 14:
		return "heat_recovery"
	case 15:
		return "cool_recovery"
	case 16:
		return "service_stop"
	default:
		return "unknown"
	}
}
