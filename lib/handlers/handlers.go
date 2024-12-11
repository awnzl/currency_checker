package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	pc "github.com/awnzl/top_currency_checker/lib/proto/pricecollector"
	rc "github.com/awnzl/top_currency_checker/lib/proto/rankcollector"
	"github.com/awnzl/top_currency_checker/lib/requester"
)

type Handlers struct {
	logger   *zap.Logger
	pcClient pc.PriceServiceClient
	rcClient rc.RankServiceClient
}

func New(log *zap.Logger, pcConn, rcConn *grpc.ClientConn) *Handlers {
	return &Handlers{
		logger: log,
		pcClient: pc.NewPriceServiceClient(pcConn),
		rcClient: rc.NewRankServiceClient(rcConn),
	}
}

func (h *Handlers) RegisterHandlers(router *mux.Router, mwFuncs ...mux.MiddlewareFunc) {
	router.HandleFunc("/", h.rootHandler)
	router.Use(mwFuncs...)
}

func (h *Handlers) rootHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	limit := 100

	if lim := r.URL.Query().Get("limit"); lim != "" {
		if limit, err = strconv.Atoi(lim); err != nil {
			h.logger.Error(err.Error())
			h.writeError("system", "invalid limit value", http.StatusBadRequest, w)
			return
		}
	}

	// get currencies rank information
	rankResp, err := h.rcClient.GetRanks(context.Background(), &rc.RankRequest{Limit: int32(limit+50)})
	if err != nil {
		h.processError(err, w)
		return
	}
	h.logger.Info("rankResp", zap.Any("currencies number", len(rankResp.List)), zap.Any("currencies", rankResp.List))

	// get prices for the currencies
	priceResp, err := h.pcClient.GetPrices(context.Background(), &pc.PriceRequest{List: rankResp.List})
	if err != nil {
		h.processError(err, w)
		return
	}
	h.logger.Info("priceResp", zap.Any("currencies number", len(priceResp.Prices)), zap.Any("currencies", priceResp.Prices))

	list := rankResp.List
	if len(rankResp.List) > limit {
		list = rankResp.List[:limit]
	}
	h.handleResponse(list, priceResp.Prices, w)
}

func (h *Handlers) processError(err error, w http.ResponseWriter) {
	if err == requester.RateLimitError {
		h.writeError("system", err.Error(), http.StatusTooManyRequests, w)
		return
	}
	h.logger.Error(err.Error())
	h.writeError("system", err.Error(), http.StatusInternalServerError, w)
}

func (h *Handlers) handleResponse(rankList []string, prices map[string]float64, w http.ResponseWriter) {
	// Rank, Symbol, Price USD
	type data struct {
		Rank int `json:"Rank"`
		Symbol string `json:"Symbol"`
		Price float64 `json:"Price USD"`
	}

	var result []data
	for rank, symbol := range rankList {
		result = append(result, data{
			Rank: rank + 1,
			Symbol: symbol,
			Price: prices[symbol],
		})
	}

	b, err := json.Marshal(result)
	if err != nil {
		h.logger.Error(err.Error())
		h.writeError("system", "internal server error", http.StatusInternalServerError, w)
		return
	}

	h.writeResponse(b, w)
}

func (h *Handlers) writeError(lvl, msg string, status int, w http.ResponseWriter) {
	w.WriteHeader(status)

	b, err := json.Marshal(
		struct {
			Level string `json:"Level,omitempty"`
			Error string `json:"Error,omitempty"`
		}{
			lvl, msg,
		},
	)
	if err != nil {
		h.logger.Error("marshal", zap.Error(err))
		return
	}

	if _, err := w.Write(b); err != nil {
		h.logger.Error("write error response", zap.Error(err))
	}
}

func (h *Handlers) writeResponse(b []byte, w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(b); err != nil {
		h.logger.Error("write response", zap.Error(err))
	}
}
