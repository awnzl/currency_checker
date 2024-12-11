package requester

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/awnzl/top_currency_checker/lib/requester/config"
)

var RateLimitError = errors.New("rate limit exceeded")

type clientAPI interface {
	Do(req *http.Request) (*http.Response, error)
}

type Requester struct {
	config     config.Config
	client     clientAPI
	limitCache map[string]time.Time
	mu         sync.Mutex
	log        *log.Logger
}

func New(config config.Config) Requester {
	return Requester{
		config: config,
		client: &http.Client{},
		limitCache: make(map[string]time.Time),
		mu: sync.Mutex{},
		log: log.New(os.Stdout, "Requester: ", log.LstdFlags | log.Lshortfile),
	}
}

func (r *Requester) checkRateLimit(req *http.Request) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := req.URL.String()
	if t, ok := r.limitCache[key]; ok {
		if int(time.Since(t).Seconds()) < r.config.RateLimit {
			return RateLimitError
		}
	}
	r.limitCache[key] = time.Now()
	return nil
}

func (r *Requester) GetData(req *http.Request) ([]byte, error) {
	if err := r.checkRateLimit(req); err != nil {
		return nil, err
	}
	return r.requestWithRetry(req)
}

func (r *Requester) requestWithRetry(req *http.Request) (data []byte, err error) {
	incomingCtx := req.Context()
	worker := func() ([]byte, error) {
		ctx, cancel := context.WithTimeout(incomingCtx, time.Duration(r.config.ReqTimeout) * time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		resp, err := r.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("send request: %w", err)
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read response body: %w", err)
		}
		return data, nil
	}

	for i := 0; i <= r.config.RetryNum; i++ {
		select {
		case <-incomingCtx.Done():
			return nil, fmt.Errorf("request canceled: %w", req.Context().Err())
		default:
			if data, err = worker(); err == nil {
				return data, nil
			}
			r.log.Printf("Request is failed, attempt #%d, error: %v", i, err.Error())
			time.Sleep(time.Duration(300) * time.Millisecond)
		}
	}

	return nil, fmt.Errorf("retry limit exceeded: %v", err)
}
