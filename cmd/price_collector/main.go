package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"google.golang.org/grpc"

	"github.com/awnzl/top_currency_checker/lib/proto/pricecollector"
	"github.com/awnzl/top_currency_checker/lib/requester/config"
	service "github.com/awnzl/top_currency_checker/lib/services/pricecollector"
)

var (
	apiKey      string
	apiURL      string
	fsymsLilmit int

	addr = "0.0.0.0:50050"
)

func prepareEnvironment() (err error) {
	apiKey = os.Getenv("api_key")
	apiURL = os.Getenv("api_endpoint")
	fsymsLilmit, err = strconv.Atoi(os.Getenv("fsymsLimit"))
	if err != nil {
		return fmt.Errorf("parse fsymsLimit: %v", err)
	}
	return nil
}

func main() {
	err := prepareEnvironment()
	if err != nil {
		log.Fatalf("Failed to prepare environment: %v\n", err)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on: %v\n", err)
	}

	config.InitConfig("./req_config.yaml")
	srv := grpc.NewServer()
	srs := service.New(service.Config{
		APIKey:     apiKey,
		APIURL:     apiURL,
		FSYMSLimit: fsymsLilmit,
		ReqConfig:  config.GetConfig(),
	})
	pricecollector.RegisterPriceServiceServer(srv, srs)

	log.Printf("Listening on %s\n", addr)
	if err := srv.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v\n", err)
	}
}
