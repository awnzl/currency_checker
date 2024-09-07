package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

const pricesURL = "https://pro-api.coinmarketcap.com/v1/cryptocurrency/listings/latest?start=1&limit=1000&convert=USD"

var apiKey string

func getCredentials() string {
	if err := godotenv.Load(); err != nil {
	  log.Fatal("Error loading .env file")
	}
	return os.Getenv("api_key")
}

func main() {
	apiKey = getCredentials() //TODO AW: do I need to return?

	list, err := getPrices()
	if err != nil {
		log.Fatalf("Failed to get prices: %v", err.Error())
	}

	fmt.Println("Prices:", list)
}

type Price struct {
	Coin string
	Price float64
}

func getPrices() ([]Price, error) {
	bts, err := requestData()
	if err != nil {
		return nil, err
	}
	return extractPrices(bts)
}

func requestData() ([]byte, error) {
	req, err := http.NewRequest("GET", pricesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Add("X-CMC_PRO_API_KEY", apiKey)

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

func extractPrices(bts []byte) ([]Price, error) {
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
		prices = append(prices, Price{Coin: each.Symbol, Price: each.Quote.USD.Price})
	}

	return prices, nil
}
