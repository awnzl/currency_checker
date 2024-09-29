package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/awnzl/top_currency_checker/lib/proto/common"
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
			return
		}
	}

	// first get rank information for the coins up to *limit*
	rankResp, err := h.rcClient.GetRanks(context.Background(), &rc.RankRequest{Limit: int32(limit)})
	if err != nil {
		//TODO AW: process error
	}

	// then based on the ranked list, get prices for the coins
	//TODO AW: change PriceRequest — you need to accept coins based on rank from the rank collector, and get prices for them
	priceResp, err := h.pcClient.GetPrices(context.Background(), &pc.PriceRequest{Limit: int32(limit)})
	if err != nil {
		//TODO AW: process error
	}

	//TODO AW: this map should be inside response instead of list of common.Price
	prices := make(map[string]float32, len(priceResp.Prices))
	for _, p := range priceResp.Prices {
		prices[p.Currency] = p.Price
	}

	result := make(map[int]common.Price, len(rankResp.Prices))
	for rank, currency := range rankResp.Prices {
		result[rank] = common.Price{Currency: currency.Currency, Price: prices[currency.Currency]}
	}

	b, err := json.Marshal(result)
	if err != nil {
		h.logger.Error(err.Error())
		h.writeError("system", "internal server error", http.StatusInternalServerError, w)
		return
	}

	if err := h.writeResponse(b, w); err != nil {
		h.logger.Error("response writing error", zap.Error(err))
	}
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
		h.logger.Error("failed to write response", zap.Error(err))
	}
}

func (h *Handlers) writeResponse(b []byte, w http.ResponseWriter) error {
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(b); err != nil {
		return err
	}

	return nil
}
