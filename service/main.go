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

	// TODO добавить конфиги для времени наблюдения, количества запросов, вайт/блек листов
	// TODO При добавлении подсети проверять в обоих списках наличие уже такой подсети
	limit := map[string]int{
		"login":    5,
		"password": 5,
		"ip":       5,
	}
	config := &bouncer.ConfigStruct{15, limit, []net.IPNet{}, []net.IPNet{}}
	service.Init(ctx, config)

	bouncer.RegisterBouncerServer(server, service)

	log.Printf("Starting server on %s", lsn.Addr().String())

	if err := server.Serve(lsn); err != nil {
		log.Fatal(err)
	}
	// TODO server.GracefulStop()
}
