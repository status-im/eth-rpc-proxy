package rpcprovider

// RpcProviderAuthType defines various authentication types for RPC providers
type RpcProviderAuthType string

const (
	NoAuth    RpcProviderAuthType = "no-auth"    // No authentication
	BasicAuth RpcProviderAuthType = "basic-auth" // HTTP Header "Authorization: Basic base64(username:password)"
	TokenAuth RpcProviderAuthType = "token-auth" // URL Token-based authentication "https://api.example.com/YOUR_TOKEN"
)

// RpcProvider represents the configuration of an RPC provider with various options
type RpcProvider struct {
	Name         string              `json:"name" validate:"required,min=1"`                                          // Provider name for identification
	URL          string              `json:"url" validate:"required,url"`                                             // URL of the current provider
	AuthType     RpcProviderAuthType `json:"authType" validate:"required,oneof=no-auth basic-auth token-auth"`        // Authentication type
	AuthLogin    string              `json:"authLogin" validate:"required_if=AuthType basic-auth,omitempty,min=1"`    // Login for BasicAuth
	AuthPassword string              `json:"authPassword" validate:"required_if=AuthType basic-auth,omitempty,min=1"` // Password for BasicAuth
	AuthToken    string              `json:"authToken" validate:"required_if=AuthType token-auth,omitempty,min=1"`    // Token for TokenAuth
}

// method unmarshal json and validate field "Enabled" exists
