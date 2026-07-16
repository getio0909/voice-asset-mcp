package backend

import (
	"context"
	"encoding/json"
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
	SupportedContractVersion = "0.1.0"
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
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: "/api/v1/system/capabilities"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Capabilities{}, fmt.Errorf("create capabilities request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(c.token) != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Capabilities{}, fmt.Errorf("request capabilities: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
		return Capabilities{}, fmt.Errorf("request capabilities: server returned HTTP %d", resp.StatusCode)
	}

	var capabilities Capabilities
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes))
	if err := decoder.Decode(&capabilities); err != nil {
		return Capabilities{}, fmt.Errorf("decode capabilities: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Capabilities{}, fmt.Errorf("decode capabilities: unexpected trailing JSON")
	}
	if err := validateCapabilities(capabilities); err != nil {
		return Capabilities{}, err
	}
	return capabilities, nil
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
