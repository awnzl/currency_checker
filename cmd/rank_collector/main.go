package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

/*
https://min-api.cryptocompare.com/data/pricemulti?fsyms=ETH,DASH&tsyms=BTC,USD,EUR&api_key=INSERT-YOUR-API-KEY-HERE
https://min-api.cryptocompare.com/data/pricemulti?fsyms=BTC,ETH,BNB,DOGE,SOL,CCL,ZXC,UKG&tsyms=USD&api_key=INSERT-YOUR-API-KEY-HERE
A current ranking information provider.

*/

//Questions:
// 1. should I collect and keep information or should I request it each time?.. requesting sounds more efficient approach...

var (
	apiKey string //TODO AW: this should be injected to the library
	apiUrl string
)

func getCredentials() (string, string) {
	if err := godotenv.Load(); err != nil {
	  log.Fatal("Error loading .env file")
	}
	return os.Getenv("api_key"), os.Getenv("endpoint")
}

func main() {
	apiKey, apiUrl = getCredentials() //TODO AW: do I actually need url in the config?..

	// create a grpc server which will request data from the provider on each client's request

	limit := 200
	list, err := getRank(limit)
	if err != nil {
		log.Fatalf("failed to get rank: %v", err.Error())
	}
	fmt.Println("~~~debug ——— rank:", list) //TODO AW: REMOVE:

//TODO AW: REMOVE:
// 1. setup and get info
// 2. output to stdout
// 3. implement proto
}

type Price struct {
	Coin  string
	Price float64
}

type response struct {
	Response   string                     `json:"Response"`
	Message    string                     `json:"Message"`
	HasWarning bool                       `json:"HasWarning"`
	Type       int                        `json:"Type"`
	RateLimit  map[string]string          `json:"RateLimit"`
	Data       map[string]json.RawMessage `json:"Data"`
}

//TODO AW: move caller to lib and use it in both services
func getRank(num int) ([]byte, error) {
	// list of all coins: https://min-api.cryptocompare.com/data/blockchain/list
	// prices of coins: https://min-api.cryptocompare.com/data/pricemulti?fsyms=BTC,ETH,BNB,DOGE,SOL,CCL,ZXC,UKG&tsyms=USD

	coins, err := getCoinsList()
	if err != nil {
		return nil, err
	}

	now := time.Now()//TODO AW: REMOVE
	coinPrices, err := getPrices(coins)
	fmt.Println("~~~debug: time:", time.Since(now)) //TODO AW: REMOVE
	fmt.Println("~~~debug: prices amount:", len(coinPrices)) //TODO AW: REMOVE
	if err != nil {
		return nil, err
	}

	sortCoinsByPrice(coinPrices)

	if len(coinPrices) > num {
		return marshalRankedCoins(coinPrices[0:num])
	}
	return marshalRankedCoins(coinPrices[0:])
}

func getCoinsList() ([]string, error) {
	responseBts, err := requestCoinsData()
	if err != nil {
		return nil, fmt.Errorf("failed to get coins data: %w", err)
	}

	return unmarshalCoinNames(responseBts)
}

func requestCoinsData() ([]byte, error) {
	return RequestGet(apiUrl+"/blockchain/list?api_key="+apiKey) //TODO AW: move url to constant (https://min-api.cryptocompare.com/data)
}

func unmarshalCoinNames(bts []byte) ([]string, error) {
	var resp response

	err := json.Unmarshal(bts, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Response != "Success" {
		return nil, fmt.Errorf("coins list response doesn't have data: %s", resp.Message)
	}

	var names []string
	for coin := range resp.Data {
		names = append(names, coin)
	}

	return names, nil
}

func getPrices(coins []string) ([]Price, error) {
	coinPrices := []Price{}
	pricesCh, errCh := requestPrices(coins)

	for {
		select {
		case prices, ok := <-pricesCh:
			coinPrices = append(coinPrices, prices...)
			if !ok {
				return coinPrices, nil
			}
		case err, ok := <-errCh:
			if ok {
				return nil, err
			}
		}
	}
}

func requestPrices(coins []string) (<-chan []Price, <-chan error) {
	// https://min-api.cryptocompare.com/data/pricemulti?fsyms=BTC,ETH,BNB,DOGE,SOL,CCL,ZXC,UKG&tsyms=USD&api_key=INSERT-YOUR-API-KEY-HERE
	pricesCh := make(chan []Price)

	const fsymsLimit = 60 // found empirically
	parts := len(coins) / fsymsLimit
	lastRangeLimit := len(coins) - parts * fsymsLimit
	errGroup, ctx := errgroup.WithContext(context.Background())

	for idx, part := 0, 0; part <= parts; part++ {
		coinsToRequest := ""
		for {
			coinsToRequest += coins[idx]+","
			idx++
			limitReached := idx % fsymsLimit == 0
			lastPartLimitReached := idx > parts * fsymsLimit && idx % fsymsLimit == lastRangeLimit
			if limitReached || lastPartLimitReached {
				coinsToRequest = strings.TrimRight(coinsToRequest, ",")
				break
			}
		}

		errGroup.Go(func() error {
			bts, err := RequestGet(apiUrl + "/pricemulti?fsyms=" + coinsToRequest + "&tsyms=USD&api_key=" + apiKey)
			if err != nil {
				return err
			}
			prices, err := unmarshalPrices(bts)
			if err != nil {
				return err
			}
			select {
			case pricesCh <- prices:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	}

	errCh := make(chan error)
	go func() {
		if err := errGroup.Wait(); err != nil {
			errCh <- err
		}
		close(errCh)
		close(pricesCh)
	}()

	return pricesCh, errCh
}

func unmarshalPrices(bts []byte) ([]Price, error) {
	// handle error response
	var errResp response
	err := json.Unmarshal(bts, &errResp)
	if err != nil {
		return nil, err
	}
	if errResp.Response == "Error" {
		return nil, fmt.Errorf("failed to get prices: %v", errResp.Message)
	}

	// handle prices
	// {"BTC":{"USD":68025.43},"ETH":{"USD":3274.18},"DOGE":{"USD":0.1313}}
	var coinsPrices map[string]struct {
		USD float64 `json:"USD"`
	}
	err = json.Unmarshal(bts, &coinsPrices)
	if err != nil {
		return nil, err
	}

	prices := []Price{}
	for coin, price := range coinsPrices {
		prices = append(prices, Price{Coin: coin, Price: price.USD})
	}

	return prices, nil
}

func sortCoinsByPrice(prices []Price) {
	slices.SortFunc(prices, func(i, j Price) int {
		if i.Price == j.Price {
			return 0
		}
		if i.Price < j.Price {
			return 1
		}
		return -1
	})
}

func marshalRankedCoins(prices []Price) ([]byte, error) {
	/* json:
		{ currencies: [
			{ rank: 1, currency: BTC, price: 65000 },
			{ rank: 2, currency: ETH, price: 5000 },
		]}
	*/
	//TODO AW: add implementation into csv or json
	fmt.Println("~~~debug: prices num:", len(prices)) //TODO AW: REMOVE:
	fmt.Println("~~~debug: coin prices:", prices) //TODO AW: REMOVE:
	return nil, nil
}

func RequestGet(uri string) ([]byte, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
