package main

import (
	"log" //TODO AW: use zap?
	"net"
	"os"

	"google.golang.org/grpc"

	pc "github.com/awnzl/top_currency_checker/lib/proto/pricecollector"
	pcService "github.com/awnzl/top_currency_checker/lib/services/pricecollector"
)

var (
	apiKey string
	apiURL string
	addr = "0.0.0.0:50050"
)

func prepareEnvironment() {
	apiKey = os.Getenv("api_key")
	apiURL = os.Getenv("api_endpoint")
}

func main() {
	prepareEnvironment()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on: %v\n", err)
	}

	srv := grpc.NewServer()
	srs := pcService.New(apiKey, apiURL)
	pc.RegisterPriceServiceServer(srv, srs)

	log.Printf("Listening on %s\n", addr)
	if err := srv.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v\n", err)
	}
}
