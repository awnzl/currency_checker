package requester

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/awnzl/top_currency_checker/lib/requester/config"
	"github.com/awnzl/top_currency_checker/lib/requester/mocks"
)

//go:generate mockgen -source=./requester.go -destination=./mocks/requester.go -package=mocks

type readCloser struct {
	Data []byte
	io.Reader
	io.Closer
}

func (r *readCloser) Close() error {
	return nil
}

func (r *readCloser) Read(p []byte) (n int, err error) {
	return copy(p, r.Data), io.EOF
}


func TestGetData(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()

	mockClient := mocks.NewMockclientAPI(c)

	resp := &http.Response{Body: &readCloser{Data: []byte("some data")}}
	mockClient.EXPECT().Do(gomock.Any()).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			time.Sleep(1300 * time.Millisecond)
			ctx := req.Context()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return resp, nil
			}
		},
	).AnyTimes()

	r := New(
		config.Config{
			ReqTimeout: 1,
			RateLimit:  5,
			RetryNum:   2,
		},
	)
	r.client = mockClient

	var req *http.Request
	req = &http.Request{
		URL: &url.URL{Path: "/data/pricemulti?fsyms=USDe,HYPE,VIRTUAL,ZBU,FLZ&tsyms=USD"},
	}
	req.WithContext(context.Background())

	data, err := r.GetData(req)
	assert.Error(t, err, "error is expected")
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.Nil(t, data, "data is expected to be nil in case of error")

	data, err = r.GetData(req)
	assert.Error(t, err, "error is expected")
	assert.Contains(t, err.Error(), "rate limit exceeded")
	assert.Nil(t, data, "data is expected to be nil in case of error")

	req = &http.Request{
		URL: &url.URL{Path: "/data/pricemulti?fsyms=FTN,GRASS,DOG,FRAX,TEL,MOODENG,SNEK,BDX&tsyms=USD"},
	}
	req.WithContext(context.Background())
	r.config.ReqTimeout = 2
	data, err = r.GetData(req)
	assert.NoError(t, err, "error is not expected")
	assert.NotNil(t, data, "data is expected to be not nil")
	assert.Equal(t, []byte("some data"), data, "data is not as expected")
}
