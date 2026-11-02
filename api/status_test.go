package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mtojek/spiroflex-vent-clear/econet"
)

const statusComponent = "comp-1"

func statusParams(overrides map[string]string) json.RawMessage {
	values := map[string]string{
		"u7074": "[1]", "u81": "[4]", "u6630": "[0]", "u6205": "[19.5]",
		"u6207": "[21.0]", "u6209": "[8.2]", "u6208": "[10.1]", "u6338": "[21.4]",
		"u6273": "[48]", "u7151": "[3]", "u6202": "[40]", "u6203": "[40]",
		"u7076": "[120]", "u6938": "[30]", "u6939": "[31]", "u6999": "[0]",
		"u6639": "[0]", "u7427": "[-1]", "u7428": "[-1]", "u7429": "[-1]",
	}
	for k, v := range overrides {
		values[k] = v
	}
	parts := make([]string, 0, len(values))
	for k, v := range values {
		parts = append(parts, `"`+k+`":`+v)
	}
	return json.RawMessage("{" + strings.Join(parts, ",") + "}")
}

func statusFrom(t *testing.T, target econet.TargetResponse) (VentStatus, error) {
	t.Helper()
	params, err := findVentParameters([]econet.TargetResponse{target}, statusComponent)
	if err != nil {
		return VentStatus{}, err
	}
	return buildVentStatus(params)
}

func TestStatusFullRead(t *testing.T) {
	st, err := statusFrom(t, econet.TargetResponse{Component: statusComponent, Parameters: statusParams(nil)})
	if err != nil {
		t.Fatal(err)
	}
	if st.Status != "normal" || st.Level == nil || *st.Level != 2 || st.SupplyTemperature != 19.5 {
		t.Fatalf("unexpected status: %+v", st)
	}
}

func TestStatusPartialReadU7151Err(t *testing.T) {
	for name, errValue := range map[string]string{"list": `["err"]`, "string": `"err"`} {
		t.Run(name, func(t *testing.T) {
			target := econet.TargetResponse{
				Component:  statusComponent,
				StatusCode: 1,
				Parameters: statusParams(map[string]string{"u7151": errValue}),
			}
			st, err := statusFrom(t, target)
			if err != nil {
				t.Fatalf("partial read must not fail: %v", err)
			}
			if st.Status != "unknown" || !st.Power || st.Humidity != 48 {
				t.Fatalf("unexpected status: %+v", st)
			}
		})
	}
}

func TestStatusRequiredParameterErrStillFails(t *testing.T) {
	target := econet.TargetResponse{
		Component:  statusComponent,
		StatusCode: 1,
		Parameters: statusParams(map[string]string{"u6205": `["err"]`}),
	}
	if _, err := statusFrom(t, target); err == nil || !strings.Contains(err.Error(), "u6205") {
		t.Fatalf("expected error naming u6205, got %v", err)
	}
}

func TestStatusTargetFailedWithoutParameters(t *testing.T) {
	target := econet.TargetResponse{Component: statusComponent, StatusCode: 1}
	if _, err := statusFrom(t, target); err == nil || !strings.Contains(err.Error(), "status code: 1") {
		t.Fatalf("expected target failure, got %v", err)
	}
}

func TestStatusPowerOffOverridesUnknown(t *testing.T) {
	target := econet.TargetResponse{
		Component:  statusComponent,
		StatusCode: 1,
		Parameters: statusParams(map[string]string{"u7074": "[0]", "u7151": `["err"]`}),
	}
	st, err := statusFrom(t, target)
	if err != nil || st.Status != "off" {
		t.Fatalf("got %+v, %v", st, err)
	}
}
