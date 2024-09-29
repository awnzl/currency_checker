package main

import (
	"log"
	"net"
	"os"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	rc "github.com/awnzl/top_currency_checker/lib/proto/rankcollector"
	rcService "github.com/awnzl/top_currency_checker/lib/services/rankcollector"
)

/*
https://min-api.cryptocompare.com/data/pricemulti?fsyms=ETH,DASH&tsyms=BTC,USD,EUR&api_key=INSERT-YOUR-API-KEY-HERE
https://min-api.cryptocompare.com/data/pricemulti?fsyms=BTC,ETH,BNB,DOGE,SOL,CCL,ZXC,UKG&tsyms=USD&api_key=INSERT-YOUR-API-KEY-HERE
A current ranking information provider.

*/

var (
	apiKey string
	apiURL string
	addr = "0.0.0.0:50051"
)

func prepareEnvironment() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
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
	srs := rcService.New(apiKey, apiURL)
	rc.RegisterRankServiceServer(srv, srs)

	log.Printf("Listening on %s\n", addr)
	if err := srv.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v\n", err)
	}
}
