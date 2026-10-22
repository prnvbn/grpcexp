package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/prnvbn/grpcexp/internal/grpc"
	"github.com/prnvbn/grpcexp/internal/tui"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	port        int
	addr        string
	protoset    string
	protoFiles  []string
	importPaths []string
	headers     []string
	useTLS      bool
	timeout     time.Duration
)

var rootCmd = &cobra.Command{
	Use:          "grpcexp",
	Short:        "grpc explorer",
	Long:         `An interactive explorer for interacting with grpc servers.`,
	RunE:         run,
	SilenceUsage: true,
}

func run(cmd *cobra.Command, args []string) error {
	var target string
	if addr != "" {
		target = addr
	} else {
		target = fmt.Sprintf("localhost:%d", port)
	}

	var creds credentials.TransportCredentials
	if useTLS {
		creds = credentials.NewTLS(&tls.Config{})
	} else {
		creds = insecure.NewCredentials()
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	grpcClient, err := grpc.NewClient(ctx, grpc.Config{
		Target:      target,
		Creds:       creds,
		UserAgent:   "grpcexp/" + strings.TrimSpace(version),
		Protoset:    protoset,
		ProtoFiles:  protoFiles,
		ImportPaths: importPaths,
		Headers:     headers,
	})
	if err != nil {
		return err
	}

	m, err := tui.NewModel(grpcClient)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().IntVarP(&port, "port", "p", 50051, "grpc server port")
	rootCmd.Flags().StringVarP(&addr, "addr", "a", "", "grpc server address")
	rootCmd.Flags().StringVar(&protoset, "protoset", "", "path to a compiled protoset descriptor file")
	rootCmd.Flags().StringArrayVar(&protoFiles, "proto", nil, "path to a .proto source file (may be repeated)")
	rootCmd.Flags().StringArrayVar(&importPaths, "import-path", nil, "directory used to resolve proto imports (may be repeated)")
	rootCmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "additional header in 'name: value' format (may be repeated)")
	rootCmd.Flags().BoolVar(&useTLS, "tls", false, "use TLS to connect to the server")
	rootCmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "connection timeout")
	rootCmd.MarkFlagsMutuallyExclusive("protoset", "proto")
}
