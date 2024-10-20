package pricecollector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	pc "github.com/awnzl/top_currency_checker/lib/proto/pricecollector"
)

type Server struct {
	pc.PriceServiceServer
	apiKey string
	apiURL string
	log    *log.Logger
}

func New(apiKey, apiURL string) *Server {
	return &Server{
		apiKey: apiKey,
		apiURL: apiURL,
		log: log.New(os.Stdout, "PriceCollector: ", log.LstdFlags | log.Lshortfile),
	}
}

// Service handler for the GetPrices RPC call
func (s *Server) GetPrices(ctx context.Context, req *pc.PriceRequest) (*pc.PriceResponse, error) {
	now := time.Now()
	// get prices for the available coins
	prices, err := s.getPrices(req.List)
	s.log.Println("Prices requesting time:", time.Since(now))
	s.log.Println("Currencies data len:", len(prices))
	if err != nil {
		return nil, err
	}

	return &pc.PriceResponse{Prices: prices}, nil
}

func (s *Server) getPrices(coins []string) (map[string]float64, error) {
	allCoinsPrices := map[string]float64{}
	pricesCh, errCh := s.requestPrices(coins)

	for prices := range pricesCh {
		for coin, price := range prices {
			allCoinsPrices[coin] = price
		}
	}

	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	default:
	}

	return allCoinsPrices, nil
}

func (s *Server) requestPrices(coins []string) (<-chan map[string]float64, <-chan error) {
	// https://min-api.cryptocompare.com/data/pricemulti?fsyms=BTC,ETH&tsyms=USD&api_key=INSERT-YOUR-API-KEY-HERE
	pricesCh := make(chan map[string]float64)
	errGroup, ctx := errgroup.WithContext(context.Background())

	const fsymsLimit = 60 // found empirically
	fullPartsNum := len(coins) / fsymsLimit
	partialPartLimit := len(coins) - fullPartsNum * fsymsLimit

	// this will collect coins in all full parts and the last, partial part, if exists
	for idx, i := 0, 0; i <= fullPartsNum; i++ {
		coinsToRequest := ""
		for {
			coinsToRequest += coins[idx]+","
			idx++
			partLimitReached := idx % fsymsLimit == 0
			partialPart := idx > fullPartsNum * fsymsLimit
			lastPartLimitReached := partialPart && idx % fsymsLimit == partialPartLimit
			if partLimitReached || lastPartLimitReached {
				coinsToRequest = strings.TrimRight(coinsToRequest, ",")
				break
			}
		}

		errGroup.Go(func() error {
			var bts []byte
			var err error
			bts, err = s.RequestGet(ctx, s.apiURL + "/pricemulti?fsyms=" + coinsToRequest + "&tsyms=USD&api_key=" + s.apiKey)
			if err != nil {
				return fmt.Errorf("requesting prices: %v", err)
			}
			prices, err := s.unmarshalPrices(bts)
			if err != nil {
				return fmt.Errorf("unmarshaling prices: %v", err)
			}

			select {
			case pricesCh <- prices:
			case <-ctx.Done():
				return fmt.Errorf("context done: %v", ctx.Err())
			}

			return nil
		})
	}

	errCh := make(chan error, 1)
	go func() {
		if err := errGroup.Wait(); err != nil {
			s.log.Println("errGroup received an error:", err.Error())
			errCh <- err
		}
		close(errCh)
		close(pricesCh)
	}()

	return pricesCh, errCh
}

func (s *Server) unmarshalPrices(bts []byte) (map[string]float64, error) {
	var errResp struct {
		Response   string `json:"Response"` // Response status: Success, Error
		Message    string `json:"Message"` // A message if Response=Error
	}
	// handle error response
	err := json.Unmarshal(bts, &errResp)
	if err != nil {
		return nil, err
	}
	if errResp.Response == "Error" {
		return nil, fmt.Errorf("getting prices: %v", errResp.Message)
	}

	// {"BTC":{"USD":68025.43},"ETH":{"USD":3274.18},"DOGE":{"USD":0.1313}}
	var coinsPrices map[string]struct {
		USD float64 `json:"USD"`
	}
	err = json.Unmarshal(bts, &coinsPrices)
	if err != nil {
		return nil, err
	}

	prices := make(map[string]float64, len(coinsPrices))
	for coin, price := range coinsPrices {
		prices[coin] = price.USD
	}

	return prices, nil
}

func (s *Server) RequestGet(ctx context.Context, uri string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
