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
)

type Handlers struct {
	logger *zap.Logger
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
			http.Error(w, "invalid limit value", http.StatusBadRequest)
			h.logger.Error(err.Error())
			h.writeError("system", "invalid limit value", http.StatusBadRequest, w)
			return
		}
	}

	// first get rank information for the coins up to *limit*
	rankResp, err := h.rcClient.GetRanks(context.Background(), &rc.RankRequest{Limit: int32(limit)})
	if err != nil {
		h.logger.Error(err.Error())
		h.writeError("system", err.Error(), http.StatusInternalServerError, w)
		return
	}

	// then based on the ranked list, get prices for the coins
	priceResp, err := h.pcClient.GetPrices(context.Background(), &pc.PriceRequest{List: rankResp.List})
	if err != nil {
		h.logger.Error(err.Error())
		h.writeError("system", err.Error(), http.StatusInternalServerError, w)
		return
	}

	h.handleResponse(rankResp.List, priceResp.Prices, w)
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
		h.logger.Error("failed to marshal", zap.Error(err))
		return
	}

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write error response", zap.Error(err))
	}
}

func (h *Handlers) writeResponse(b []byte, w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", zap.Error(err))
	}
}
