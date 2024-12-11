package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"github.com/awnzl/top_currency_checker/lib/proto/rankcollector"
	"github.com/awnzl/top_currency_checker/lib/requester/config"
	service "github.com/awnzl/top_currency_checker/lib/services/rankcollector"
)

var (
	apiKey string
	apiURL string
	addr = "0.0.0.0:50051"
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

	config.InitConfig("./req_config.yaml")
	srv := grpc.NewServer()
	srs := service.New(service.Config{
		APIKey:    apiKey,
		APIURL:    apiURL,
		ReqConfig: config.GetConfig(),
	})
	rankcollector.RegisterRankServiceServer(srv, srs)

	log.Printf("Listening on %s\n", addr)
	if err := srv.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v\n", err)
	}
}
