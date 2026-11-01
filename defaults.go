package spiroflex

import (
	"fmt"
	"strings"
)

// Public identifiers of the Spiroflex ecoNET cloud backend. They are the same
// for every user (they are embedded in the official mobile app), so only the
// account username and password need to be configured. Each value can still be
// overridden in the configuration in case the vendor changes its backend.
const (
	DefaultRegion         = "eu-central-1"
	DefaultUserPoolID     = "eu-central-1_que8JCEVH"
	DefaultClientID       = "3qga0ntgrj7fblcgau24plp2h4"
	DefaultIdentityPoolID = "eu-central-1:d1f8d4ae-7401-4d69-b603-1e992f635253"
	DefaultGatewayName    = "6tx17a8ijh"
	DefaultIoTName        = "a24t7r3f2r1nrr-ats"
	DefaultAPIEndpoint    = "0.0.0.0:8088"
)

// ApplyDefaults fills empty backend settings with the built-in Spiroflex
// values.
//
// When none of the backend identifiers are configured, the region is also
// reset to the default. This protects configurations created from older
// templates that shipped a placeholder region together with empty identifiers.
func (c *Config) ApplyDefaults() {
	trim := func(s *string) { *s = strings.TrimSpace(*s) }
	for _, s := range []*string{
		&c.Region, &c.Cognito.Username, &c.Cognito.UserPoolID, &c.Cognito.ClientID,
		&c.Cognito.IdentityPoolID, &c.Gateway.Name, &c.IoT.Name,
		&c.Installation.ID, &c.Installation.Name, &c.API.Endpoint,
	} {
		trim(s)
	}

	backendUnset := c.Cognito.UserPoolID == "" && c.Cognito.ClientID == "" &&
		c.Cognito.IdentityPoolID == "" && c.Gateway.Name == "" && c.IoT.Name == ""
	if backendUnset || c.Region == "" {
		c.Region = DefaultRegion
	}

	setIfEmpty := func(s *string, v string) {
		if *s == "" {
			*s = v
		}
	}
	setIfEmpty(&c.Cognito.UserPoolID, DefaultUserPoolID)
	setIfEmpty(&c.Cognito.ClientID, DefaultClientID)
	setIfEmpty(&c.Cognito.IdentityPoolID, DefaultIdentityPoolID)
	setIfEmpty(&c.Gateway.Name, DefaultGatewayName)
	setIfEmpty(&c.IoT.Name, DefaultIoTName)
	setIfEmpty(&c.API.Endpoint, DefaultAPIEndpoint)
}

// Validate reports missing settings that have no built-in default.
func (c *Config) Validate() error {
	var missing []string
	if c.Cognito.Username == "" {
		missing = append(missing, "cognito.username")
	}
	if c.Cognito.Password == "" {
		missing = append(missing, "cognito.password")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required settings: %s (use the Spiroflex mobile app login)", strings.Join(missing, ", "))
	}
	return nil
}
