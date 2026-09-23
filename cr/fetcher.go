package cr

import (
	"context"
	"io"
	"log"
	"mime"
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

func (f *Fetcher) Fetch(ctx context.Context, pageUrl string) FetchResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageUrl, nil)
	if err != nil {
		f.log.Printf("Error creating request url=%d | error:= %s", pageUrl, err)
		return FetchResult{}
	}
	resp, err := f.client.Do(req)
	if err != nil {
		f.log.Printf("Error fetching url=%d | error:%s", pageUrl, err)
		return FetchResult{}
	}
	defer resp.Body.Close()
	f.log.Printf("Fetched url=%d | statusCode=%d", pageUrl, resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return FetchResult{}
	}
	if !isHtml(resp.Header.Get("Content-Type")) {
		f.log.Printf("Skip url=%d | error:%s", pageUrl, resp.Header.Get("Content-Type"))
		return FetchResult{}
	}
	title, links, err := Parse(pageUrl, resp.Body)
	if err != nil {
		f.log.Printf("error url=%s err=%v", pageUrl, err)
		return FetchResult{}
	}
	return FetchResult{
		Title: title,
		Links: links,
		OK:    true,
	}
}

func isHtml(head string) bool {
	mediaType, _, err := mime.ParseMediaType(head)
	return err == nil && mediaType == "text/html"
}
