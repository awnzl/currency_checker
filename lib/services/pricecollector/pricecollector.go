package pricecollector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/awnzl/top_currency_checker/lib/proto/common"
	pc "github.com/awnzl/top_currency_checker/lib/proto/pricecollector"
)

//TODO AW: add tests

const uriParamFormat = "start=1&limit=%d&convert=USD"

type Server struct {
	pc.PriceServiceServer
	apiKey string
	apiURL string
}

func New(apiKey, apiURL string) *Server {
	return &Server{
		apiKey: apiKey,
		apiURL: apiURL,
	}
}

//TODO AW: change PriceRequest — you need to accept coins based on rank from the rank collector, and get prices for them
func (srv *Server) GetPrices(ctx context.Context, req *pc.PriceRequest) (*pc.PriceResponse, error) {
	// this endpoint returns cryptocurrency in order of CoinMarketCap's market cap rank
	uri := fmt.Sprintf(srv.apiURL+uriParamFormat, req.Limit)

	bts, err := srv.requestData(uri)
	if err != nil {
		return nil, err
	}

	prices, err := srv.extractPrices(bts)
	if err != nil {
		return nil, err
	}

	return srv.GetResponse(prices), nil
}

//TODO AW: should this be a part of proto? how does client should know about the json layout?
type Price struct {
	Currency string `json:"currency,omitempty"`
	Price float64 `json:"price,omitempty"`
}

func (srv *Server) requestData(uri string) ([]byte, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create a request: %w", err)
	}
	req.Header.Add("X-CMC_PRO_API_KEY", srv.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failure: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response reading failed: %w", err)
	}

	return data, nil
}

func (srv *Server) extractPrices(bts []byte) ([]Price, error) {
	type responseData struct {
		Data []struct {
			Symbol string `json:"symbol"`
			Quote  struct {
				USD struct {
					Price float64 `json:"price"`
				} `json:"USD"`
			} `json:"quote"`
		} `json:"data"`
		Status struct {
			ErrCode int `json:"error_code"`
			ErrMsg string `json:"error_message"`
		} `json:"status"`
	}

	var resp responseData
	err := json.Unmarshal(bts, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Status.ErrCode != 0 {
		return nil, fmt.Errorf("request error: %v", resp.Status.ErrMsg)
	}

	var prices []Price
	for _, each := range resp.Data {
		prices = append(prices, Price{Currency: each.Symbol, Price: each.Quote.USD.Price})
	}

	return prices, nil
}

func (srv *Server) GetResponse(prices []Price) *pc.PriceResponse {
	var data []*common.Price

	for _, p := range prices {
		data = append(data, &common.Price{Currency: p.Currency, Price: float32(p.Price)})
	}

	return &pc.PriceResponse{Prices: data}
}
