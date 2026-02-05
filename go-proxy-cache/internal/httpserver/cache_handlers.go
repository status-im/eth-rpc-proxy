package httpserver

import (
	"fmt"
	"net/http"

	"github.com/status-im/proxy-common/models"
)

// handleGet handles GET cache requests
func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	var req CacheRequest
	if err := s.parseRequest(r, &req); err != nil {
		s.writeErrorResponse(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Chain == "" || req.Network == "" || req.RawBody == "" {
		s.writeErrorResponse(w, "Missing required fields: chain, network, raw_body", http.StatusBadRequest)
		return
	}

	// Use cache service to handle the get operation
	result, err := s.cacheService.Get(req.Chain, req.Network, req.RawBody)
	if err != nil {
		s.writeErrorResponse(w, fmt.Sprintf("Cache service error: %v", err), http.StatusBadRequest)
		return
	}

	// Determine cache status
	var cacheStatus models.CacheStatus
	if result.Bypass {
		cacheStatus = models.CacheStatusBypass
	} else if result.Found {
		cacheStatus = models.CacheStatusHit
	} else {
		cacheStatus = models.CacheStatusMiss
	}

	s.writeResponse(w, &CacheResponse{
		Success:     true,
		Found:       result.Found,
		Fresh:       result.Fresh,
		Data:        result.Data,
		Key:         result.Key,
		CacheType:   result.CacheType,
		TTL:         result.TTL,
		CacheStatus: cacheStatus,
		CacheLevel:  result.CacheLevel,
	})
}

// handleSet handles SET cache requests
func (s *Server) handleSet(w http.ResponseWriter, r *http.Request) {
	var req CacheRequest
	if err := s.parseRequest(r, &req); err != nil {
		s.writeErrorResponse(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Chain == "" || req.Network == "" || req.RawBody == "" || req.Data == "" {
		s.writeErrorResponse(w, "Missing required fields: chain, network, raw_body, data", http.StatusBadRequest)
		return
	}

	// Use cache service to handle the set operation
	err := s.cacheService.Set(req.Chain, req.Network, req.RawBody, req.Data, req.TTL, req.StaleTTL)
	if err != nil {
		s.writeErrorResponse(w, fmt.Sprintf("Cache service error: %v", err), http.StatusBadRequest)
		return
	}

	s.writeResponse(w, &CacheResponse{
		Success: true,
	})
}

// handleCacheInfo handles cache info requests (equivalent to cache rules check)
func (s *Server) handleCacheInfo(w http.ResponseWriter, r *http.Request) {
	var req CacheRequest
	if err := s.parseRequest(r, &req); err != nil {
		s.writeErrorResponse(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Chain == "" || req.Network == "" || req.RawBody == "" {
		s.writeErrorResponse(w, "Missing required fields: chain, network, raw_body", http.StatusBadRequest)
		return
	}

	// Use cache service to get cache info
	cacheType, ttl, err := s.cacheService.GetCacheInfo(req.Chain, req.Network, req.RawBody)
	if err != nil {
		s.writeErrorResponse(w, fmt.Sprintf("Cache service error: %v", err), http.StatusBadRequest)
		return
	}

	s.writeResponse(w, &CacheResponse{
		Success:   true,
		CacheType: cacheType,
		TTL:       ttl,
	})
}
