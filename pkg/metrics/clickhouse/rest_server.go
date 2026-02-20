// Copyright 2019 Altinity Ltd and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clickhouse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/altinity/clickhouse-operator/pkg/apis/metrics"
)

// Request type constants
const (
	RequestTypeCR   = "cr"
	RequestTypeHost = "host"
)

// RESTRequest wraps different request types for POST/DELETE operations
type RESTRequest struct {
	Type string             `json:"type"` // "cr" or "host"
	CR   *metrics.WatchedCR `json:"cr,omitempty"`
	Host *HostRequest       `json:"host,omitempty"`
}

// HostRequest contains host details with parent context
type HostRequest struct {
	CRNamespace string               `json:"crNamespace"`
	CRName      string               `json:"crName"`
	ClusterName string               `json:"clusterName"`
	Host        *metrics.WatchedHost `json:"host"`
}

// IsValid checks if HostRequest has all required fields
func (r *HostRequest) IsValid() bool {
	return r.CRNamespace != "" && r.CRName != "" && r.ClusterName != "" && r.Host != nil && r.Host.Hostname != ""
}

// RESTServer provides HTTP API for managing watched CRs and Hosts
type RESTServer struct {
	registry *CRRegistry
}

// NewRESTServer creates a new RESTServer instance
func NewRESTServer(registry *CRRegistry) *RESTServer {
	return &RESTServer{
		registry: registry,
	}
}

// ServeHTTP implements http.Handler interface
func (s *RESTServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/chi" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGet(w, r)
	case http.MethodPost:
		s.handlePost(w, r)
	case http.MethodDelete:
		s.handleDelete(w, r)
	default:
		_, _ = fmt.Fprintf(w, "Sorry, only GET, POST and DELETE methods are supported.")
	}
}

// handleGet serves HTTP GET request to get list of watched CRs
func (s *RESTServer) handleGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.registry.List())
}

// handlePost serves HTTP POST request to add CR or Host
func (s *RESTServer) handlePost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	req, err := s.decodeRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	switch req.Type {
	case RequestTypeCR:
		s.registry.AddCR(req.CR)
	case RequestTypeHost:
		if err := s.registry.AddHost(req.Host); err != nil {
			http.Error(w, err.Error(), http.StatusNotAcceptable)
			return
		}
	default:
		http.Error(w, fmt.Sprintf("unknown request type: %s", req.Type), http.StatusNotAcceptable)
	}
}

// handleDelete serves HTTP DELETE request to delete CR or Host
func (s *RESTServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	req, err := s.decodeRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	switch req.Type {
	case RequestTypeCR:
		s.registry.RemoveCR(req.CR)
	case RequestTypeHost:
		s.registry.RemoveHost(req.Host)
	default:
		http.Error(w, fmt.Sprintf("unknown request type: %s", req.Type), http.StatusNotAcceptable)
	}
}

// decodeRequest decodes RESTRequest from the HTTP request body
func (s *RESTServer) decodeRequest(r *http.Request) (*RESTRequest, error) {
	req := &RESTRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return nil, fmt.Errorf("unable to parse request: %w", err)
	}

	switch req.Type {
	case RequestTypeCR:
		if req.CR == nil || !req.CR.IsValid() {
			return nil, fmt.Errorf("invalid CR in request")
		}
	case RequestTypeHost:
		if req.Host == nil || !req.Host.IsValid() {
			return nil, fmt.Errorf("invalid Host in request")
		}
	default:
		return nil, fmt.Errorf("unknown request type: %s", req.Type)
	}

	return req, nil
}

// StartMetricsREST starts Prometheus metrics exporter and REST API server
func StartMetricsREST(
	metricsAddress string,
	metricsPath string,
	collectorTimeout time.Duration,
	chiListAddress string,
	chiListPath string,
) *Exporter {
	log.V(1).Infof("Starting metrics exporter at '%s%s'\n", metricsAddress, metricsPath)

	// Create shared registry
	registry := NewCRRegistry()

	// Create and register Prometheus exporter
	exporter := NewExporter(registry, collectorTimeout)
	prometheus.MustRegister(exporter)

	// Create REST server
	restServer := NewRESTServer(registry)

	// Setup HTTP handlers
	http.Handle(metricsPath, promhttp.Handler())
	http.Handle(chiListPath, restServer)

	// Start HTTP servers
	go http.ListenAndServe(metricsAddress, nil)
	if metricsAddress != chiListAddress {
		go http.ListenAndServe(chiListAddress, nil)
	}

	return exporter
}
