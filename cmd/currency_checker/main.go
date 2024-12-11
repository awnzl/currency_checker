/*
A price list service for top crypto assets.

The service should expose an HTTP endpoint, which when fetched, displays an up-to-date list of top assets and their current prices in USD.
* The endpoint should support `limit` parameter which indicates how many top coins should be returned.
* The output should be either `JSON` or `CSV` compatible.

Create command line client for both API (top and score)
* parallel requests (you can specify how many goroutines and requests per threads from command line)
* you can specify which API to request from command line
* routines have to return errors to main goroutine before error exit, and main goroutine print it to terminal
* pretty print or json output

Example call should look somehow like this:

```
$ curl http://localhost:6667?limit=200

Rank,	Symbol,	Price USD,
1,	BTC,	6634.41,
2,	ETH,	370.237,
3,	XRP,	0.471636,
...	...	...
200,	DCN,	0.000269788,
```
*/
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/awnzl/top_currency_checker/lib/handlers"
	"github.com/awnzl/top_currency_checker/lib/logger"
	"github.com/awnzl/top_currency_checker/lib/middleware"
)

const (
	logLevel = "info"
	port = "8080"
)

var log *zap.Logger

var (
	pcAddr = os.Getenv("PC_ADDRESS")
	rcAddr = os.Getenv("RC_ADDRESS")
)

// connects to the service or exits the program if the connection can't be established
func getConnection(addr string) *grpc.ClientConn {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("can't establish connection to service", zap.String("address", addr), zap.Error(err))
	}
	log.Info("connected", zap.String("address", addr))
	return conn
}

func main() {
	log = logger.NewZap(logLevel)
	defer log.Sync()

	pcConn := getConnection(pcAddr)
	defer pcConn.Close()

	rcConn := getConnection(rcAddr)
	defer rcConn.Close()

	router := mux.NewRouter()
	handlers.New(log, pcConn, rcConn).RegisterHandlers(
		router,
		middleware.NewMiddlewareLogger(log).Log,
		middleware.SetContentTypeJSON,
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: router,
	}

	go func() {
		log.Info("start listening", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", zap.Error(err))
			os.Exit(1)
		}
	}()

	<-sigChan
	log.Info("Received an interrupt signal...")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("failure during server shutdown: %w", zap.Error(err))
	}
	log.Info("Server stopped...")
}
