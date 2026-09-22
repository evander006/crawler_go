package cr

import (
	"io"
	"log"
	"net/http"
	"time"
)

type FetchResult struct {
	Title string
	Links []string
	OK    bool
}

type Fetcher struct {
	log    *log.Logger
	client *http.Client
}

func NewFetcher(requestTimeOut time.Duration, logger *log.Logger) *Fetcher {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &Fetcher{
		log: logger,
		client: &http.Client{
			Timeout: requestTimeOut,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}
