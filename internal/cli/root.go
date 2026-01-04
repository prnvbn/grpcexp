package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/prnvbn/grpcexp/internal/grpc"
	"github.com/prnvbn/grpcexp/internal/tui"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	port     int
	addr     string
	protoset string
)

var rootCmd = &cobra.Command{
	Use:          "grpcexp",
	Short:        "grpc explorer",
	Long:         `An interactive explorer for interacting with grpc servers that implement reflection - https://grpc.io/docs/guides/reflection/`,
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcClient, err := grpc.NewClient(ctx, grpc.Config{
		Target:    target,
		Creds:     insecure.NewCredentials(), //todo: make configureable
		UserAgent: "grpcexp/" + version,
		Protoset:  protoset,
	})
	if err != nil {
		return err
	}

	m, err := tui.NewModel(grpcClient)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
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
	rootCmd.Flags().StringVar(&protoset, "protoset", "", "path to protoset file (uses server reflection if not specified)")
}
