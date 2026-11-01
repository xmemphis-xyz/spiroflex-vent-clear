package econet

import (
	"fmt"
	"strings"
)

// SelectInstallation picks the installation to control from the list returned
// by the ecoNET API.
//
// Installations without access are ignored when the API reports access
// explicitly. A single remaining installation is always selected, so a stale
// or default name in the configuration does not block startup. With several
// installations, wantName (case-insensitive) chooses one; otherwise an error
// lists the available names and IDs so the user can configure one.
func SelectInstallation(list []Installation, wantName string) (Installation, error) {
	candidates := make([]Installation, 0, len(list))
	for _, inst := range list {
		if inst.HasAccess {
			candidates = append(candidates, inst)
		}
	}
	if len(candidates) == 0 {
		candidates = list
	}

	switch len(candidates) {
	case 0:
		return Installation{}, fmt.Errorf("no installations found for this ecoNET account")
	case 1:
		return candidates[0], nil
	}

	wantName = strings.TrimSpace(wantName)
	if wantName != "" {
		for _, inst := range candidates {
			if strings.EqualFold(strings.TrimSpace(inst.Name), wantName) {
				return inst, nil
			}
		}
	}

	names := make([]string, 0, len(candidates))
	for _, inst := range candidates {
		names = append(names, fmt.Sprintf("%q (id %s)", inst.Name, inst.ID))
	}
	return Installation{}, fmt.Errorf(
		"account has %d installations; set installation.name or installation.id to one of: %s",
		len(candidates), strings.Join(names, ", "))
}
