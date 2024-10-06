package rankcollector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	rc "github.com/awnzl/top_currency_checker/lib/proto/rankcollector"
)

const uriParamFormat = "start=1&limit=%d&convert=USD"

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

// Service handler for the GetRanks RPC call
func (srv *Server) GetRanks(ctx context.Context, req *rc.RankRequest) (*rc.RankResponse, error) {
	// this endpoint returns cryptocurrencies in order of CoinMarketCap's market cap rank
	uri := fmt.Sprintf(srv.apiURL+uriParamFormat, req.Limit)

	bts, err := srv.requestData(uri)
	if err != nil {
		return nil, err
	}

	data, err := srv.extractRanks(bts)
	if err != nil {
		return nil, err
	}

	return &rc.RankResponse{List: data}, nil
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
