package api

import (
	"context"
	"fmt"
	"log"

	"github.com/mtojek/spiroflex-vent-clear/econet"
)

// installation returns the ecoNET installation to control. A configured
// installation ID is used as-is; otherwise the installation is discovered from
// the account once and cached. A failed lookup is retried on the next request.
func (ws *WebServer) installation(ctx context.Context, client *econet.Client) (econet.Installation, error) {
	if ws.c.Installation.ID != "" {
		return econet.Installation{ID: ws.c.Installation.ID, Name: ws.c.Installation.Name}, nil
	}

	ws.instMu.Lock()
	defer ws.instMu.Unlock()

	if ws.inst != nil {
		return *ws.inst, nil
	}

	list, err := client.Installations(ctx)
	if err != nil {
		return econet.Installation{}, fmt.Errorf("unable to discover installation: %w", err)
	}
	inst, err := econet.SelectInstallation(list, ws.c.Installation.Name)
	if err != nil {
		return econet.Installation{}, err
	}
	log.Printf("Using installation %q (id %s, connected=%t)", inst.Name, inst.ID, inst.IsConnected)
	ws.inst = &inst
	return inst, nil
}

// DiscoverInstallation logs in and resolves the installation, so that
// configuration problems are reported in the log right after startup instead
// of on the first API request.
func (ws *WebServer) DiscoverInstallation(ctx context.Context) error {
	client, err := econet.New(ctx, ws.c)
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}
	_, err = ws.installation(ctx, client)
	return err
}
