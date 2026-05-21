package grpc

import (
	"testing"

	"google.golang.org/grpc/credentials/insecure"
)

func TestGRPCURLCommand(t *testing.T) {
	client := &Client{
		config: Config{
			Target:    "localhost:50051",
			Creds:     insecure.NewCredentials(),
			UserAgent: "grpcexp/test",
			Protoset:  "api fixtures/echo.protoset",
		},
	}

	got, err := client.GRPCURLCommand("echo.v1.EchoService.Echo", map[string]any{
		"message": "it's here",
	})
	if err != nil {
		t.Fatalf("GRPCURLCommand returned error: %v", err)
	}

	want := `grpcurl -plaintext -protoset 'api fixtures/echo.protoset' -user-agent grpcexp/test -d '{"message":"it'"'"'s here"}' localhost:50051 echo.v1.EchoService.Echo`
	if got != want {
		t.Fatalf("GRPCURLCommand = %q, want %q", got, want)
	}
}
