package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	port int
	addr string
)

var rootCmd = &cobra.Command{
	Use:   "grpcexp",
	Short: "grpc explorer",
	Long:  `An interactive explorer for interacting with grpc servers that implement reflection - https://grpc.io/docs/guides/reflection/`,
	RunE:  run,
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

	// todo: set this based on the flags
	var creds credentials.TransportCredentials

	var opts []grpc.DialOption
	grpcexpUA := "grpcexp/" + version
	opts = append(opts, grpc.WithUserAgent(grpcexpUA))
	cc, err := grpcurl.BlockingDial(ctx, "", target, creds, opts...)
	if err != nil {
		return err
	}

	refClient := grpcreflect.NewClientAuto(context.Background(), cc)
	refClient.AllowMissingFileDescriptors()

	services, err := refClient.ListServices()
	if err != nil {
		return err
	}

	for _, service := range services {
		fmt.Println(service)
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
}
