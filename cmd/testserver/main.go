package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"

	echov1 "github.com/prnvbn/grpcexp/cmd/testserver/echo"
	hellov1 "github.com/prnvbn/grpcexp/cmd/testserver/hello"
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
	hellov1.UnimplementedHelloServiceServer
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

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	helloworldpb.RegisterGreeterServer(s, &server{})
	echov1.RegisterEchoServiceServer(s, &server{})
	hellov1.RegisterHelloServiceServer(s, &server{})

	reflection.Register(s)
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
