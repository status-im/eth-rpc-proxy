package testutils

import (
	"fmt"
	"net/http"
	"sync"
)

// ProviderSetup manages multiple mock RPC servers
type ProviderSetup struct {
	servers []httpServer
	wg      sync.WaitGroup
}

type httpServer interface {
	Start() error
	Stop() error
}

// NewProviderSetup creates a new provider setup
func NewProviderSetup() *ProviderSetup {
	return &ProviderSetup{
		servers: make([]httpServer, 0),
	}
}

// AddProvider adds a new mock provider
func (p *ProviderSetup) AddProvider(port int, responses map[string]map[string]interface{}) *MockRPCServer {
	server := NewMockRPCServer(port)
	for method, response := range responses {
		server.AddResponse(method, response)
	}
	p.servers = append(p.servers, server)
	return server
}

// Add404Provider adds a provider that returns 404 for all requests
func (p *ProviderSetup) Add404Provider(port int) {
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}),
	}
	p.servers = append(p.servers, &http404Server{server})
}

// StartAll starts all mock providers
func (p *ProviderSetup) StartAll() error {
	for _, server := range p.servers {
		p.wg.Add(1)
		go func(s httpServer) {
			defer p.wg.Done()
			s.Start()
		}(server)
	}
	return nil
}

// StopAll stops all mock providers
func (p *ProviderSetup) StopAll() error {
	for _, server := range p.servers {
		if err := server.Stop(); err != nil {
			return err
		}
	}
	p.wg.Wait()
	return nil
}

// http404Server wraps http.Server to implement httpServer interface
type http404Server struct {
	*http.Server
}

func (s *http404Server) Start() error {
	return s.ListenAndServe()
}

func (s *http404Server) Stop() error {
	return s.Close()
}
