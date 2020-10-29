package main

import (
	"context"
	"log"
	"net"

	bouncer "../pkg"
	"google.golang.org/grpc"
)

func main() {
	lsn, err := net.Listen("tcp", "localhost:50051")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	server := grpc.NewServer()
	service := &bouncer.Service{}
	service.Init(ctx, 15, []string{}, []string{})

	bouncer.RegisterBouncerServer(server, service)

	log.Printf("Starting server on %s", lsn.Addr().String())

	if err := server.Serve(lsn); err != nil {
		log.Fatal(err)
	}
	// TODO server.GracefulStop()
}
