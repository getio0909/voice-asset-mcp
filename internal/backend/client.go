package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

const maxResponseBytes = 1 << 20

const (
	SupportedAPIVersion      = "v1"
	SupportedContractVersion = "0.22.0"
)

var featureNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type Capabilities struct {
	ServerVersion   string   `json:"server_version"`
	APIVersion      string   `json:"api_version"`
	ContractVersion string   `json:"contract_version"`
	Features        []string `json:"features"`
}

type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
}

func NewClient(rawBaseURL, token string, httpClient *http.Client) (*Client, error) {
	baseURL, err := url.Parse(rawBaseURL)
	if err != nil || baseURL.Host == "" || (baseURL.Scheme != "http" && baseURL.Scheme != "https") {
		return nil, fmt.Errorf("base URL must be an absolute HTTP(S) URL")
	}
	if baseURL.User != nil {
		return nil, fmt.Errorf("base URL must not contain credentials")
	}
	if baseURL.Scheme == "http" && !isLoopback(baseURL.Hostname()) {
		return nil, fmt.Errorf("base URL must use HTTPS for non-loopback hosts")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{baseURL: baseURL, token: token, httpClient: httpClient}, nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *Client) GetSystemCapabilities(ctx context.Context) (Capabilities, error) {
	var capabilities Capabilities
	if err := c.getJSON(ctx, "/api/v1/system/capabilities", nil, &capabilities, "request capabilities"); err != nil {
		return Capabilities{}, err
	}
	if err := validateCapabilities(capabilities); err != nil {
		return Capabilities{}, err
	}
	return capabilities, nil
}

func (c *Client) getJSON(
	ctx context.Context,
	path string,
	query url.Values,
	destination any,
	operation string,
) error {
	return c.doJSON(ctx, http.MethodGet, path, query, nil, nil, http.StatusOK, destination, operation)
}

func (c *Client) writeJSON(
	ctx context.Context,
	method,
	path string,
	payload any,
	headers map[string]string,
	expectedStatus int,
	destination any,
	operation string,
) error {
	return c.doJSON(ctx, method, path, nil, payload, headers, expectedStatus, destination, operation)
}

func (c *Client) doJSON(
	ctx context.Context,
	method,
	path string,
	query url.Values,
	payload any,
	headers map[string]string,
	expectedStatus int,
	destination any,
	operation string,
) error {
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: path, RawQuery: query.Encode()})
	var requestBody io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("%s: encode request: %w", operation, err)
		}
		requestBody = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), requestBody)
	if err != nil {
		return fmt.Errorf("%s: create request: %w", operation, err)
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(c.token) != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("%s: read response: %w", operation, err)
	}
	if len(responseBody) > maxResponseBytes {
		return fmt.Errorf("%s: response exceeds %d bytes", operation, maxResponseBytes)
	}
	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("%s: server returned HTTP %d", operation, resp.StatusCode)
	}

	decoder := json.NewDecoder(bytes.NewReader(responseBody))
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("%s: decode response: %w", operation, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s: decode response: unexpected trailing JSON", operation)
	}
	return nil
}

func validateCapabilities(capabilities Capabilities) error {
	if strings.TrimSpace(capabilities.ServerVersion) == "" {
		return fmt.Errorf("validate capabilities: server_version is required")
	}
	if capabilities.APIVersion != SupportedAPIVersion {
		return fmt.Errorf("validate capabilities: unsupported api_version %q", capabilities.APIVersion)
	}
	if capabilities.ContractVersion != SupportedContractVersion {
		return fmt.Errorf("validate capabilities: unsupported contract_version %q", capabilities.ContractVersion)
	}
	if capabilities.Features == nil {
		return fmt.Errorf("validate capabilities: features is required")
	}
	if !sort.StringsAreSorted(capabilities.Features) {
		return fmt.Errorf("validate capabilities: features must be sorted")
	}
	seen := make(map[string]struct{}, len(capabilities.Features))
	for _, feature := range capabilities.Features {
		if !featureNamePattern.MatchString(feature) {
			return fmt.Errorf("validate capabilities: invalid feature name %q", feature)
		}
		if _, exists := seen[feature]; exists {
			return fmt.Errorf("validate capabilities: duplicate feature %q", feature)
		}
		seen[feature] = struct{}{}
	}
	return nil
}
