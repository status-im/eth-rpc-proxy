package provider

// RPCProviderAuthType defines various authentication types for RPC providers
type RPCProviderAuthType string

const (
	NoAuth    RPCProviderAuthType = "no-auth"    // No authentication
	BasicAuth RPCProviderAuthType = "basic-auth" // HTTP Header "Authorization: Basic base64(username:password)"
	TokenAuth RPCProviderAuthType = "token-auth" // URL Token-based authentication "https://api.example.com/YOUR_TOKEN"
)

// RPCProvider represents the configuration of an RPC provider with various options
type RPCProvider struct {
	Name         string              `json:"name" validate:"required,min=1"`                                          // Provider name for identification
	URL          string              `json:"url" validate:"required,url"`                                             // URL of the current provider
	AuthType     RPCProviderAuthType `json:"authType" validate:"required,oneof=no-auth basic-auth token-auth"`        // Authentication type
	AuthLogin    string              `json:"authLogin" validate:"required_if=AuthType basic-auth,omitempty,min=1"`    // Login for BasicAuth
	AuthPassword string              `json:"authPassword" validate:"required_if=AuthType basic-auth,omitempty,min=1"` // Password for BasicAuth
	AuthToken    string              `json:"authToken" validate:"required_if=AuthType token-auth,omitempty,min=1"`    // Token for TokenAuth
	ChainID      int64               `json:"chain_id"`
}

// method unmarshal json and validate field "Enabled" exists
