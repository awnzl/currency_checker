package rankcollector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	rc "github.com/awnzl/top_currency_checker/lib/proto/rankcollector"
	"github.com/awnzl/top_currency_checker/lib/requester"
	"github.com/awnzl/top_currency_checker/lib/requester/config"
)

const uriParamFormat = "start=1&limit=%d&convert=USD"

type Config struct {
	APIKey     string
	APIURL     string
	ReqConfig  config.Config
}

type Server struct {
	rc.RankServiceServer
	requester requester.Requester
	apiKey string
	apiURL string
}

func New(conf Config) *Server {
	return &Server{
		requester: requester.New(conf.ReqConfig),
		apiKey:    conf.APIKey,
		apiURL:    conf.APIURL,
	}
}

// Service handler for the GetRanks RPC call
func (srv *Server) GetRanks(ctx context.Context, req *rc.RankRequest) (*rc.RankResponse, error) {
	// this endpoint returns cryptocurrencies in order of CoinMarketCap's market cap rank
	uri := fmt.Sprintf(srv.apiURL+uriParamFormat, req.Limit)

	bts, err := srv.requestData(ctx, uri)
	if err != nil {
		return nil, err
	}

	data, err := srv.extractRanks(bts)
	if err != nil {
		return nil, err
	}

	return &rc.RankResponse{List: data}, nil
}

func (srv *Server) requestData(ctx context.Context, uri string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("create a request: %w", err)
	}
	req.Header.Add("X-CMC_PRO_API_KEY", srv.apiKey)

	return srv.requester.GetData(req)
}

func (srv *Server) extractRanks(bts []byte) ([]string, error) {
	type responseData struct {
		Data []struct { // If no errors, the response will contain an array of objects
			Symbol string `json:"symbol"`
		} `json:"data"`
		Status struct { // If there is an error, the response will contain an object with error details
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

	data := []string{}
	for _, each := range resp.Data {
		data = append(data, each.Symbol)
	}

	return data, nil
}
