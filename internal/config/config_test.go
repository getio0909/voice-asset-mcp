package config

import "testing"

func TestValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:   "stdio defaults",
			config: Config{Transport: TransportStdio, ServerBaseURL: "http://127.0.0.1:8080"},
		},
		{
			name:   "loopback HTTP without inbound token",
			config: Config{Transport: TransportHTTP, ListenAddr: "127.0.0.1:8090", ServerBaseURL: "https://server.example"},
		},
		{
			name:    "remote HTTP requires inbound token",
			config:  Config{Transport: TransportHTTP, ListenAddr: "0.0.0.0:8090", ServerBaseURL: "https://server.example"},
			wantErr: true,
		},
		{
			name:    "reject URL credentials",
			config:  Config{Transport: TransportStdio, ServerBaseURL: "https://user:secret@server.example"},
			wantErr: true,
		},
		{
			name:    "reject insecure remote Server URL",
			config:  Config{Transport: TransportStdio, ServerBaseURL: "http://server.example"},
			wantErr: true,
		},
		{
			name: "remote HTTP requires native TLS",
			config: Config{
				Transport:       TransportHTTP,
				ListenAddr:      "0.0.0.0:8090",
				ServerBaseURL:   "https://server.example",
				HTTPBearerToken: "inbound-token",
			},
			wantErr: true,
		},
		{
			name: "remote HTTP with bearer and TLS",
			config: Config{
				Transport:       TransportHTTP,
				ListenAddr:      "0.0.0.0:8090",
				ServerBaseURL:   "https://server.example",
				HTTPBearerToken: "inbound-token",
				TLSCertFile:     "server.crt",
				TLSKeyFile:      "server.key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
