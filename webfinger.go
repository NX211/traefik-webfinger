// Package traefik_webfinger implements a Traefik middleware for handling WebFinger requests.
package traefik_webfinger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Define static errors.
var (
	ErrDomainRequired      = errors.New("domain must be specified")
	ErrResourceDomainMatch = errors.New("resource does not match configured domain")
	ErrSubjectRequired     = errors.New("subject is required for resource")
	ErrRelRequired         = errors.New("rel is required for links in resource")
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
	Domain string `json:"domain,omitempty" yaml:"domain"`
	// Default resources and their links
	Resources map[string]WebFingerResponse `json:"resources,omitempty" yaml:"resources"`
	// Whether to pass through to the backend service if resource not found
	Passthrough bool `json:"passthrough,omitempty" yaml:"passthrough"`
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
		return nil, ErrDomainRequired
	}

	// Validate resources
	for resource, response := range config.Resources {
		if !isResourceForDomain(resource, config.Domain) {
			return nil, fmt.Errorf("%w: %s for domain %s", ErrResourceDomainMatch, resource, config.Domain)
		}

		if response.Subject == "" {
			return nil, fmt.Errorf("%w: %s", ErrSubjectRequired, resource)
		}

		for _, link := range response.Links {
			if link.Rel == "" {
				return nil, fmt.Errorf("%w: %s", ErrRelRequired, resource)
			}
		}
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
func (w *WebFinger) ServeHTTP(responseWriter http.ResponseWriter, req *http.Request) {
	// Only handle WebFinger requests to the well-known path
	if !strings.HasPrefix(req.URL.Path, "/.well-known/webfinger") {
		w.next.ServeHTTP(responseWriter, req)
		return
	}

	// WebFinger only works with GET requests
	if req.Method != http.MethodGet {
		http.Error(responseWriter, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the resource parameter
	resource := req.URL.Query().Get("resource")
	if resource == "" {
		http.Error(responseWriter, "Resource parameter is required", http.StatusBadRequest)
		return
	}

	// Check if the resource belongs to the configured domain
	if !isResourceForDomain(resource, w.domain) {
		if w.passthrough {
			w.next.ServeHTTP(responseWriter, req)
			return
		}

		http.Error(responseWriter, "Resource not found", http.StatusNotFound)

		return
	}

	// If the resource is specified in our configuration, return it
	if response, exists := w.resources[resource]; exists {
		responseWriter.Header().Set("Content-Type", "application/jrd+json")
		responseWriter.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(responseWriter).Encode(response); err != nil {
			http.Error(responseWriter, "Error encoding response", http.StatusInternalServerError)
			return
		}

		return
	}

	// If passthrough is enabled, forward the request to the backend
	if w.passthrough {
		w.next.ServeHTTP(responseWriter, req)
		return
	}

	// Otherwise, return a 404
	http.Error(responseWriter, "Resource not found", http.StatusNotFound)
}

// isResourceForDomain checks if the resource belongs to the configured domain.
func isResourceForDomain(resource, domain string) bool {
	// Resource can be in different formats, most commonly:
	// acct:user@example.com, https://example.com/user, or mailto:user@example.com

	const (
		acctPrefix      = "acct:"
		acctPrefixLen   = len(acctPrefix)
		httpsPrefix     = "https://"
		httpsPrefixLen  = len(httpsPrefix)
		mailtoPrefix    = "mailto:"
		mailtoPrefixLen = len(mailtoPrefix)
		splitLimit      = 2
	)

	if strings.HasPrefix(resource, acctPrefix) {
		parts := strings.SplitN(resource[acctPrefixLen:], "@", splitLimit)
		return len(parts) == splitLimit && parts[1] == domain
	}

	if strings.HasPrefix(resource, httpsPrefix) {
		return strings.Contains(resource[httpsPrefixLen:], domain)
	}

	if strings.HasPrefix(resource, mailtoPrefix) {
		parts := strings.SplitN(resource[mailtoPrefixLen:], "@", splitLimit)
		return len(parts) == splitLimit && parts[1] == domain
	}

	// For other resource types, check if the domain is part of the resource
	return strings.Contains(resource, domain)
}
