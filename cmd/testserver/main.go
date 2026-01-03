package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	helloworldpb "google.golang.org/grpc/examples/helloworld/helloworld"
	routeguidepb "google.golang.org/grpc/examples/route_guide/routeguide"
	"google.golang.org/grpc/reflection"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

type server struct {
	helloworldpb.UnimplementedGreeterServer
	routeguidepb.UnimplementedRouteGuideServer
}

func (s *server) SayHello(_ context.Context, in *helloworldpb.HelloRequest) (*helloworldpb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &helloworldpb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	// Greeter service (helloworld)
	helloworldpb.RegisterGreeterServer(s, &server{})

	// RouteGuide service - has streaming RPCs and complex message types
	// Unary: GetFeature, Server streaming: ListFeatures,
	// Client streaming: RecordRoute, Bidirectional streaming: RouteChat
	routeguidepb.RegisterRouteGuideServer(s, &server{})

	reflection.Register(s)
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
