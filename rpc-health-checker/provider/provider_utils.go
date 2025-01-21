package provider

import (
	"encoding/json"
	"os"

	"github.com/go-playground/validator/v10"
)

// RPCProvidersFile describes the structure of the root JSON file for providers.
type RPCProvidersFile struct {
	Providers []RPCProvider `json:"providers" validate:"required,dive"` // List of providers
}

// ReadRpcProviders reads the list of providers from a JSON file with validation.
func ReadRpcProviders(filename string) ([]RPCProvider, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pf RPCProvidersFile
	if err := json.NewDecoder(f).Decode(&pf); err != nil {
		return nil, err
	}

	// Validate providers
	validate := validator.New()
	if err := validate.Struct(pf); err != nil {
		return nil, err
	}

	return pf.Providers, nil
}

// WriteRpcProviders writes the list of providers to a JSON file with validation.
func WriteRpcProviders(filename string, providers []RPCProvider) error {
	// Validate providers before writing
	validate := validator.New()
	pf := RPCProvidersFile{
		Providers: providers,
	}
	if err := validate.Struct(pf); err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ") // For readability
	if err := encoder.Encode(pf); err != nil {
		return err
	}

	return nil
}
