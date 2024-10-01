package rankcollector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/awnzl/top_currency_checker/lib/proto/common"
	rc "github.com/awnzl/top_currency_checker/lib/proto/rankcollector"
	"golang.org/x/sync/errgroup"
)

//TODO AW: add tests

type Server struct {
	rc.RankServiceServer
	apiKey string
	apiURL string
}

func New(apiKey, apiURL string) *Server {
	return &Server{
		apiKey: apiKey,
		apiURL: apiURL,
	}
}

type Price struct {
	Currency  string
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

//TODO AW: implement:
func (s *Server) GetRanks(ctx context.Context, req *rc.RankRequest) (*rc.RankResponse, error) {
	//TODO AW: REMOVE
	// list of all coins: https://min-api.cryptocompare.com/data/blockchain/list
	// prices of coins: https://min-api.cryptocompare.com/data/pricemulti?fsyms=BTC,ETH,BNB,DOGE,SOL,CCL,ZXC,UKG&tsyms=USD


	//TODO AW: !!!!!!!!! change the ranking model — use market capitalization:
	// https://min-api.cryptocompare.com/data/pricemultifull?fsyms=BTC,ETH,BNB,DOGE,SOL,CCL,ZXC,UKG&tsyms=USD
	// Object
    // RAW: Object
    //     BTC: Object
    //         USD: Object
    //             ...
    //             MKTCAP: 1301943747738.8972


	// get list of available coins
	coins, err := s.getCoinsList()
	if err != nil {
		return nil, err
	}

	now := time.Now()//TODO AW: REMOVE

	//TODO AW: get market capitalization here!!
	// get prices for the available coins
	coinPrices, err := s.getPrices(coins)
	fmt.Println("~~~debug: time:", time.Since(now)) //TODO AW: REMOVE
	fmt.Println("~~~debug: prices amount:", len(coinPrices)) //TODO AW: REMOVE
	if err != nil {
		return nil, err
	}

	//TODO AW: you should use market capitalization here
	// sort coins to get rank list
	s.sortCoinsByPrice(coinPrices)

	if len(coinPrices) > int(req.Limit) {
		return s.getResponse(coinPrices[0:req.Limit]), nil
	}
	return s.getResponse(coinPrices[0:]), nil
}
//TODO AW: REMOVE:
// func getRank(num int) ([]byte, error) {
// 	// list of all coins: https://min-api.cryptocompare.com/data/blockchain/list
// 	// prices of coins: https://min-api.cryptocompare.com/data/pricemulti?fsyms=BTC,ETH,BNB,DOGE,SOL,CCL,ZXC,UKG&tsyms=USD

// 	coins, err := getCoinsList()
// 	if err != nil {
// 		return nil, err
// 	}

// 	now := time.Now()//TODO AW: REMOVE
// 	coinPrices, err := getPrices(coins)
// 	fmt.Println("~~~debug: time:", time.Since(now)) //TODO AW: REMOVE
// 	fmt.Println("~~~debug: prices amount:", len(coinPrices)) //TODO AW: REMOVE
// 	if err != nil {
// 		return nil, err
// 	}

// 	sortCoinsByPrice(coinPrices)

// 	if len(coinPrices) > num {
// 		return marshalRankedCoins(coinPrices[0:num])
// 	}
// 	return marshalRankedCoins(coinPrices[0:])
// }

func (s *Server) getCoinsList() ([]string, error) {
	responseBts, err := s.RequestGet(s.apiURL+"/blockchain/list?api_key="+s.apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get coins data: %w", err)
	}

	return s.unmarshalCoinNames(responseBts)
}

func (s *Server) unmarshalCoinNames(bts []byte) ([]string, error) {
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

//TODO AW: change function to get market capitalization
func (s *Server) getPrices(coins []string) ([]Price, error) {
	//TODO AW: !!!!!!!!! change the ranking model — use market capitalization:
	// https://min-api.cryptocompare.com/data/pricemultifull?fsyms=BTC,ETH,BNB,DOGE,SOL,CCL,ZXC,UKG&tsyms=USD
	// Object
    // RAW: Object
    //     BTC: Object
    //         USD: Object
    //             ...
    //             MKTCAP: 1301943747738.8972

	coinPrices := []Price{}
	pricesCh, errCh := s.requestPrices(coins)

	for prices := range pricesCh {
		coinPrices = append(coinPrices, prices...)
	}

	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	default:
	}

	return coinPrices, nil
}

func (s *Server) requestPrices(coins []string) (<-chan []Price, <-chan error) {
	// https://min-api.cryptocompare.com/data/pricemulti?fsyms=BTC,ETH,BNB,DOGE,SOL,CCL,ZXC,UKG&tsyms=USD&api_key=INSERT-YOUR-API-KEY-HERE
	pricesCh := make(chan []Price)
	errGroup, ctx := errgroup.WithContext(context.Background())

	const fsymsLimit = 60 // found empirically
	fullPartsNum := len(coins) / fsymsLimit
	partialPartLimit := len(coins) - fullPartsNum * fsymsLimit

	// this will iterate through all full parts and the last, partial part, if exists
	for idx, i := 0, 0; i <= fullPartsNum; i++ {
		errGroup.Go(func() error {
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

			bts, err := s.RequestGet(s.apiURL + "/pricemulti?fsyms=" + coinsToRequest + "&tsyms=USD&api_key=" + s.apiKey)
			if err != nil {
				return err
			}

			prices, err := s.unmarshalPrices(bts)
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

func (s *Server) unmarshalPrices(bts []byte) ([]Price, error) {
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
		prices = append(prices, Price{Currency: coin, Price: price.USD})
	}

	return prices, nil
}

func (s *Server) sortCoinsByPrice(prices []Price) {
	slices.SortFunc(
		prices,
		func(i, j Price) int {
			if i.Price == j.Price {
				return 0
			}
			if i.Price < j.Price {
				return 1
			}
			return -1
		},
	)
}

func (s *Server) getResponse(prices []Price) *rc.RankResponse {
	fmt.Println("~~~debug: prices num:", len(prices)) //TODO AW: REMOVE:
	fmt.Println("~~~debug: coin prices:", prices) //TODO AW: REMOVE:

	var res []*common.Price
	for _, p := range prices {
		res = append(res, &common.Price{Currency: p.Currency, Price: float32(p.Price)})
	}

	return &rc.RankResponse{Prices: res}
}

func (s *Server) RequestGet(uri string) ([]byte, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
