package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	diagnosticsv1 "github.com/prnvbn/grpcexp/cmd/testserver/diagnostics"
	echov1 "github.com/prnvbn/grpcexp/cmd/testserver/echo"
	hellov1 "github.com/prnvbn/grpcexp/cmd/testserver/hello"
	helloworldpb "google.golang.org/grpc/examples/helloworld/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

type server struct {
	diagnosticsv1.UnimplementedDiagnosticsServiceServer
	helloworldpb.UnimplementedGreeterServer
	echov1.UnimplementedEchoServiceServer
	hellov1.UnimplementedHelloServiceServer
}

func (s *server) InspectUnary(ctx context.Context, in *diagnosticsv1.InspectUnaryRequest) (*diagnosticsv1.InspectUnaryResponse, error) {
	if err := grpc.SendHeader(ctx, metadata.Pairs(
		"x-grpcexp-test-header", "response-header-value",
		"x-grpcexp-test-header", "response-header-value-2",
	)); err != nil {
		return nil, status.Errorf(codes.Internal, "send response headers: %v", err)
	}
	grpc.SetTrailer(ctx, metadata.Pairs("x-grpcexp-test-trailer", "response-trailer-value"))

	if !in.GetReturnError() {
		return &diagnosticsv1.InspectUnaryResponse{Message: "unary response with metadata"}, nil
	}

	st, err := status.New(codes.InvalidArgument, "structured error requested").WithDetails(
		&diagnosticsv1.FailureDetail{
			Reason: "RETURN_ERROR_ENABLED",
			Metadata: map[string]string{
				"field": "return_error",
				"hint":  "set return_error to false for a successful response",
			},
		},
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "attach structured error details: %v", err)
	}
	return nil, st.Err()
}

func (s *server) SayHello(_ context.Context, in *helloworldpb.HelloRequest) (*helloworldpb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &helloworldpb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func (s *server) Echo(_ context.Context, in *echov1.Message) (*echov1.Message, error) {
	log.Printf("Received: %v", in.GetMessage())
	return in, nil
}

func (s *server) EchoStream(stream echov1.EchoService_EchoStreamServer) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		log.Printf("Received stream: %v", msg.GetMessage())
		if err := stream.Send(msg); err != nil {
			return err
		}
	}
}

func (s *server) HelloStream(stream hellov1.HelloService_HelloStreamServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		name := req.GetName()
		if name == "" {
			name = "there"
		}
		log.Printf("Received hello stream: %v", name)
		if err := stream.Send(&hellov1.HelloReply{Message: "Hello " + name}); err != nil {
			return err
		}
	}
}

func (s *server) SlowHello(ctx context.Context, in *hellov1.SlowHelloRequest) (*hellov1.SlowHelloResponse, error) {
	duration := in.GetDuration()
	if duration == nil {
		return nil, status.Error(codes.InvalidArgument, "duration is required")
	}
	if err := duration.CheckValid(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid duration: %v", err)
	}

	delay := duration.AsDuration()
	if delay < 0 {
		return nil, status.Error(codes.InvalidArgument, "duration must not be negative")
	}

	name := in.GetName()
	if name == "" {
		name = "there"
	}
	log.Printf("Received slow hello: %v, waiting %s", name, delay)

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil, status.FromContextError(ctx.Err()).Err()
	case <-timer.C:
		return &hellov1.SlowHelloResponse{Message: "Hello " + name}, nil
	}
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	diagnosticsv1.RegisterDiagnosticsServiceServer(s, &server{})
	helloworldpb.RegisterGreeterServer(s, &server{})
	echov1.RegisterEchoServiceServer(s, &server{})
	hellov1.RegisterHelloServiceServer(s, &server{})

	reflection.Register(s)
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
