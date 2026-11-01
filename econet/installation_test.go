package econet

import (
	"strings"
	"testing"
)

func TestSelectInstallationSingle(t *testing.T) {
	list := []Installation{{ID: "a", Name: "SCP V", HasAccess: true}}
	// A stale/default name must not block a single-installation account.
	got, err := SelectInstallation(list, "ecoVENT Simple")
	if err != nil || got.ID != "a" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestSelectInstallationIgnoresNoAccess(t *testing.T) {
	list := []Installation{
		{ID: "a", Name: "Old", HasAccess: false},
		{ID: "b", Name: "Home", HasAccess: true},
	}
	got, err := SelectInstallation(list, "")
	if err != nil || got.ID != "b" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestSelectInstallationAccessFieldMissing(t *testing.T) {
	list := []Installation{{ID: "a", Name: "Home"}}
	got, err := SelectInstallation(list, "")
	if err != nil || got.ID != "a" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestSelectInstallationByName(t *testing.T) {
	list := []Installation{
		{ID: "a", Name: "Dom", HasAccess: true},
		{ID: "b", Name: "Biuro", HasAccess: true},
	}
	got, err := SelectInstallation(list, " biuro ")
	if err != nil || got.ID != "b" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestSelectInstallationAmbiguous(t *testing.T) {
	list := []Installation{
		{ID: "a", Name: "Dom", HasAccess: true},
		{ID: "b", Name: "Biuro", HasAccess: true},
	}
	_, err := SelectInstallation(list, "")
	if err == nil || !strings.Contains(err.Error(), `"Biuro" (id b)`) {
		t.Fatalf("expected ambiguity error listing installations, got %v", err)
	}
}

func TestSelectInstallationEmpty(t *testing.T) {
	if _, err := SelectInstallation(nil, ""); err == nil {
		t.Fatal("expected error")
	}
}
