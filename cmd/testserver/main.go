package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	echov1 "github.com/prnvbn/grpcexp/cmd/testserver/echo"
	helloworldpb "google.golang.org/grpc/examples/helloworld/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

type server struct {
	helloworldpb.UnimplementedGreeterServer
	echov1.UnimplementedEchoServiceServer
}

func (s *server) SayHello(_ context.Context, in *helloworldpb.HelloRequest) (*helloworldpb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &helloworldpb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func (s *server) Echo(_ context.Context, in *echov1.Message) (*echov1.Message, error) {
	log.Printf("Received: %v", in.GetMessage())
	return in, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	helloworldpb.RegisterGreeterServer(s, &server{})
	echov1.RegisterEchoServiceServer(s, &server{})

	reflection.Register(s)
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
