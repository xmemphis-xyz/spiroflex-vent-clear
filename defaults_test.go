package spiroflex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyDefaultsFillsBackend(t *testing.T) {
	c := Config{Region: "eu-west-3"} // placeholder from older templates
	c.ApplyDefaults()
	if c.Region != DefaultRegion || c.Cognito.UserPoolID != DefaultUserPoolID ||
		c.Cognito.ClientID != DefaultClientID || c.Cognito.IdentityPoolID != DefaultIdentityPoolID ||
		c.Gateway.Name != DefaultGatewayName || c.IoT.Name != DefaultIoTName ||
		c.API.Endpoint != DefaultAPIEndpoint {
		t.Fatalf("defaults not applied: %+v", c)
	}
}

func TestApplyDefaultsKeepsOverrides(t *testing.T) {
	c := Config{Region: "eu-west-1"}
	c.Cognito.UserPoolID = "eu-west-1_custom"
	c.ApplyDefaults()
	if c.Region != "eu-west-1" || c.Cognito.UserPoolID != "eu-west-1_custom" {
		t.Fatalf("overrides lost: %+v", c)
	}
	if c.Cognito.ClientID != DefaultClientID {
		t.Fatalf("missing field not defaulted: %+v", c)
	}
}

func TestValidateRequiresCredentials(t *testing.T) {
	c := Config{}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for missing credentials")
	}
	c.Cognito.Username, c.Cognito.Password = "u", "p"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfigMinimal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("cognito:\n  username: user@example.com\n  password: secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SPIROFLEX_CONFIG_FILE", path)
	c, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !c.API.Rest || c.API.Endpoint != DefaultAPIEndpoint || c.Gateway.Name != DefaultGatewayName {
		t.Fatalf("unexpected config: %+v", c)
	}
}

func TestLoadConfigHomeAssistantOptions(t *testing.T) {
	// options.json as written by the HA app with empty optional fields.
	path := filepath.Join(t.TempDir(), "options.json")
	body := `{"cognito":{"username":"u","password":"p","user_pool_id":""},"gateway":{"name":""},` +
		`"installation":{"name":"","id":""},"api":{"endpoint":"0.0.0.0:8088","rest":true}}`
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SPIROFLEX_CONFIG_FILE", path)
	c, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if c.Cognito.UserPoolID != DefaultUserPoolID || c.Installation.ID != "" {
		t.Fatalf("unexpected config: %+v", c)
	}
}
