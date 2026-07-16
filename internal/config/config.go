package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
)

type Config struct {
	Transport       string
	ListenAddr      string
	ServerBaseURL   string
	ServerToken     string
	HTTPBearerToken string
	TLSCertFile     string
	TLSKeyFile      string
}

func FromEnv() Config {
	return Config{
		Transport:       envOrDefault("VOICE_ASSET_MCP_TRANSPORT", TransportStdio),
		ListenAddr:      envOrDefault("VOICE_ASSET_MCP_LISTEN", "127.0.0.1:8090"),
		ServerBaseURL:   envOrDefault("VOICE_ASSET_SERVER_URL", "http://127.0.0.1:8080"),
		ServerToken:     os.Getenv("VOICE_ASSET_SERVER_TOKEN"),
		HTTPBearerToken: os.Getenv("VOICE_ASSET_MCP_HTTP_TOKEN"),
		TLSCertFile:     os.Getenv("VOICE_ASSET_MCP_TLS_CERT_FILE"),
		TLSKeyFile:      os.Getenv("VOICE_ASSET_MCP_TLS_KEY_FILE"),
	}
}

func (c Config) Validate() error {
	if c.Transport != TransportStdio && c.Transport != TransportHTTP {
		return fmt.Errorf("transport must be %q or %q", TransportStdio, TransportHTTP)
	}
	u, err := url.Parse(c.ServerBaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("VOICE_ASSET_SERVER_URL must be an absolute HTTP(S) URL")
	}
	if u.User != nil {
		return fmt.Errorf("VOICE_ASSET_SERVER_URL must not contain credentials")
	}
	if u.Scheme == "http" && !isLoopback(u.Hostname()) {
		return fmt.Errorf("VOICE_ASSET_SERVER_URL must use HTTPS for non-loopback hosts")
	}
	if c.Transport == TransportHTTP {
		host, _, err := net.SplitHostPort(c.ListenAddr)
		if err != nil {
			return fmt.Errorf("VOICE_ASSET_MCP_LISTEN must include host and port: %w", err)
		}
		if !isLoopback(host) && strings.TrimSpace(c.HTTPBearerToken) == "" {
			return fmt.Errorf("VOICE_ASSET_MCP_HTTP_TOKEN is required for non-loopback HTTP")
		}
		if (c.TLSCertFile == "") != (c.TLSKeyFile == "") {
			return fmt.Errorf("VOICE_ASSET_MCP_TLS_CERT_FILE and VOICE_ASSET_MCP_TLS_KEY_FILE must be set together")
		}
		if !isLoopback(host) && c.TLSCertFile == "" {
			return fmt.Errorf("TLS certificate and key files are required for non-loopback HTTP")
		}
	}
	return nil
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
