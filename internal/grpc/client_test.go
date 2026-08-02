package grpc

import (
	"testing"

	"google.golang.org/grpc/credentials/insecure"
)

func TestValidateHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		wantErr bool
	}{
		{name: "none"},
		{name: "multiple", headers: []string{"authorization: Bearer token", "x-label: value:with:colons"}},
		{name: "empty value", headers: []string{"x-debug:"}},
		{name: "missing colon", headers: []string{"authorization"}, wantErr: true},
		{name: "empty name", headers: []string{" : value"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHeaders(tt.headers)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateHeaders() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGRPCURLCommand(t *testing.T) {
	client := &Client{
		config: Config{
			Target:    "localhost:50051",
			Creds:     insecure.NewCredentials(),
			UserAgent: "grpcexp/test",
			Protoset:  "api fixtures/echo.protoset",
			Headers: []string{
				"authorization: Bearer test-token",
				"x-request-label: it's a test",
			},
		},
	}

	got, err := client.GRPCURLCommand("echo.v1.EchoService.Echo", map[string]any{
		"message": "it's here",
	})
	if err != nil {
		t.Fatalf("GRPCURLCommand returned error: %v", err)
	}

	want := `grpcurl -plaintext -protoset 'api fixtures/echo.protoset' -user-agent grpcexp/test -H 'authorization: Bearer test-token' -H 'x-request-label: it'"'"'s a test' -d '{"message":"it'"'"'s here"}' localhost:50051 echo.v1.EchoService.Echo`
	if got != want {
		t.Fatalf("GRPCURLCommand = %q, want %q", got, want)
	}
}
