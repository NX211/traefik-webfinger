// Package webfinger implements a Traefik middleware for handling WebFinger requests.
package webfinger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// WebFingerResponse represents the WebFinger JSON response according to RFC 7033.
type WebFingerResponse struct {
	Subject string          `json:"subject"`
	Aliases []string        `json:"aliases,omitempty"`
	Links   []WebFingerLink `json:"links,omitempty"`
}

// WebFingerLink represents a link in the WebFinger response.
type WebFingerLink struct {
	Rel        string            `json:"rel"`
	Type       string            `json:"type,omitempty"`
	Href       string            `json:"href,omitempty"`
	Titles     map[string]string `json:"titles,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// Config defines the plugin configuration structure.
type Config struct {
	// The domain this WebFinger service is responsible for
	Domain string `json:"domain,omitempty"`
	// Default resources and their links
	Resources map[string]WebFingerResponse `json:"resources,omitempty"`
	// Whether to pass through to the backend service if resource not found
	Passthrough bool `json:"passthrough,omitempty"`
}

// CreateConfig creates a new default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Domain:      "",
		Resources:   make(map[string]WebFingerResponse),
		Passthrough: false,
	}
}

// WebFinger is the middleware plugin implementation.
type WebFinger struct {
	next        http.Handler
	name        string
	domain      string
	resources   map[string]WebFingerResponse
	passthrough bool
}

// New creates a new WebFinger middleware plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.Domain == "" {
		return nil, fmt.Errorf("domain must be specified")
	}
	
	return &WebFinger{
		next:        next,
		name:        name,
		domain:      config.Domain,
		resources:   config.Resources,
		passthrough: config.Passthrough,
	}, nil
}

// ServeHTTP implements the http.Handler interface.
func (w *WebFinger) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Only handle WebFinger requests to the well-known path
	if !strings.HasPrefix(req.URL.Path, "/.well-known/webfinger") {
		w.next.ServeHTTP(rw, req)
		return
	}

	// WebFinger only works with GET requests
	if req.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the resource parameter
	resource := req.URL.Query().Get("resource")
	if resource == "" {
		http.Error(rw, "Resource parameter is required", http.StatusBadRequest)
		return
	}

	// Check if the resource belongs to the configured domain
	if !isResourceForDomain(resource, w.domain) {
		http.Error(rw, "Resource not found", http.StatusNotFound)
		return
	}

	// If the resource is specified in our configuration, return it
	normalizedResource := normalizeResource(resource)
	if response, exists := w.resources[normalizedResource]; exists {
		rw.Header().Set("Content-Type", "application/jrd+json")
		rw.WriteHeader(http.StatusOK)
		
		if err := json.NewEncoder(rw).Encode(response); err != nil {
			http.Error(rw, "Error encoding response", http.StatusInternalServerError)
		}
		return
	}

	// If passthrough is enabled, forward the request to the backend
	if w.passthrough {
		w.next.ServeHTTP(rw, req)
		return
	}

	// Otherwise, return a 404
	http.Error(rw, "Resource not found", http.StatusNotFound)
}

// isResourceForDomain checks if the resource belongs to the configured domain.
func isResourceForDomain(resource, domain string) bool {
	// Resource can be in different formats, most commonly:
	// acct:user@example.com, https://example.com/user, or mailto:user@example.com
	
	if strings.HasPrefix(resource, "acct:") {
		parts := strings.SplitN(resource[5:], "@", 2)
		return len(parts) == 2 && parts[1] == domain
	}
	
	if strings.HasPrefix(resource, "https://") {
		return strings.Contains(resource[8:], domain)
	}
	
	if strings.HasPrefix(resource, "mailto:") {
		parts := strings.SplitN(resource[7:], "@", 2)
		return len(parts) == 2 && parts[1] == domain
	}
	
	// For other resource types, check if the domain is part of the resource
	return strings.Contains(resource, domain)
}

// normalizeResource returns a normalized version of the resource identifier.
func normalizeResource(resource string) string {
	// This is a simplistic normalization that just returns the resource as-is.
	// In a production environment, you might want to implement more sophisticated
	// normalization, such as handling case insensitivity for email-like identifiers.
	return resource
}