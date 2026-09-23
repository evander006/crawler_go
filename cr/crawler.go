// cr/crawler.go
package cr

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
)

const workerCount = 10

type task struct {
	url    string
	depth  int
	host   string
	parent *Page
	rootIx int
}

type Crawler struct {
	fetcher *Fetcher
}

func NewCrawler(fetcher *Fetcher) *Crawler {
	return &Crawler{fetcher: fetcher}
}

type crawlRun struct {
	fetcher *Fetcher
	ctx     context.Context
	jobs    chan task
	quit    chan struct{}
	once    sync.Once
	pending atomic.Int64
	mu      sync.Mutex
	visited map[string]struct{}
	roots   []*Page
}

func (c *Crawler) Crawl(ctx context.Context, urls []string, depth int) []*Page {
	if depth < 0 {
		depth = 0
	}
	r := &crawlRun{
		fetcher: c.fetcher,
		ctx:     ctx,
		jobs:    make(chan task, workerCount),
		quit:    make(chan struct{}),
		visited: make(map[string]struct{}),
		roots:   make([]*Page, len(urls)),
	}
	for i := 0; i < workerCount; i++ {
		go r.worker()
	}
	for i, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			r.fetcher.log.Printf("error url=%s err=%v", raw, err)
			continue
		}
		if !r.tryVisit(raw) {
			continue
		}
		r.enqueue(task{url: raw, depth: depth, host: u.Host, rootIx: i})
	}
	if r.pending.Load() == 0 {
		r.closeQuit()
	}
	<-r.quit
	return r.pages()
}

func (r *crawlRun) worker() {
	for {
		select {
		case <-r.quit:
			return
		case t := <-r.jobs:
			r.handle(t)
			r.finish()
		}
	}
}

func (r *crawlRun) handle(t task) {
	if r.ctx.Err() != nil {
		return
	}
	res := r.fetcher.Fetch(r.ctx, t.url)
	if !res.OK {
		return
	}
	node := NewPage(t.url, res.Title)
	r.mu.Lock()
	if t.parent == nil {
		r.roots[t.rootIx] = node
	} else {
		t.parent.Links = append(t.parent.Links, node)
	}
	r.mu.Unlock()

	if t.depth == 0 {
		return
	}
	for _, link := range res.Links {
		if r.ctx.Err() != nil {
			return
		}
		if !sameHost(link, t.host) || !r.tryVisit(link) {
			continue
		}
		r.enqueue(task{
			url:    link,
			depth:  t.depth - 1,
			host:   t.host,
			parent: node,
		})
	}
}

func (r *crawlRun) enqueue(t task) {
	r.pending.Add(1)
	select {
	case <-r.ctx.Done():
		r.finish()
	case r.jobs <- t:
	default:
		go func() {
			select {
			case <-r.ctx.Done():
				r.finish()
			case r.jobs <- t:
			}
		}()
	}
}

func (r *crawlRun) finish() {
	if r.pending.Add(-1) == 0 {
		r.closeQuit()
	}
}

func (r *crawlRun) closeQuit() {
	r.once.Do(func() { close(r.quit) })
}

func (r *crawlRun) tryVisit(raw string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.visited[raw]; ok {
		return false
	}
	r.visited[raw] = struct{}{}
	return true
}

func sameHost(raw, host string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, host)
}

func (r *crawlRun) pages() []*Page {
	out := make([]*Page, 0, len(r.roots))
	for _, page := range r.roots {
		if page != nil {
			out = append(out, page)
		}
	}
	return out
}
