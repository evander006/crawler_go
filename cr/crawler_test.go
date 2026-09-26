package cr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	body := `<html><title>  Hello  </title><a href="next">n</a><a href="mailto:a@b.c">m</a><a href="/abs">a</a>`
	title, links, err := Parse("https://example.com/dir/page", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if title != "Hello" {
		t.Fatalf("title = %q", title)
	}
	want := []string{"https://example.com/dir/next", "https://example.com/abs"}
	if len(links) != len(want) || links[0] != want[0] || links[1] != want[1] {
		t.Fatalf("links = %#v", links)
	}

	data, err := json.Marshal(NewPage("https://example.com", "Example"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"links":[]`) {
		t.Fatalf("json = %s", data)
	}
}

func TestFetchHTMLAndSkips(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/page":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<title>Hi</title><a href="/a">a</a>`)
		case "/go":
			http.Redirect(w, r, "/page", http.StatusFound)
		case "/plain":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "text")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	f := NewFetcher(time.Second, log.New(&buf, "", 0))
	ctx := context.Background()

	got := f.Fetch(ctx, srv.URL+"/page")
	if !got.OK || got.Title != "Hi" || len(got.Links) != 1 {
		t.Fatalf("html = %+v", got)
	}
	if f.Fetch(ctx, srv.URL+"/go").OK {
		t.Fatal("redirect was accepted")
	}
	if f.Fetch(ctx, srv.URL+"/plain").OK {
		t.Fatal("plain text was accepted")
	}
	if f.Fetch(ctx, srv.URL+"/missing").OK {
		t.Fatal("404 was accepted")
	}
	logText := buf.String()
	for _, status := range []string{"statusCode=200", "statusCode=302", "statusCode=404"} {
		if !strings.Contains(logText, status) {
			t.Fatalf("log %q, missing %s", logText, status)
		}
	}
}

func TestCrawlDepthHostAndCycle(t *testing.T) {
	var home, deep, deeper atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			home.Add(1)
			fmt.Fprint(w, `<title>Home</title>
				<a href="/a">a</a><a href="/a">again</a>
				<a href="/missing">m</a>
				<a href="https://other.test/x">x</a>
				<a href="/b">b</a>`)
		case "/a":
			fmt.Fprint(w, `<title>A</title><a href="/">back</a><a href="/deep">d</a>`)
		case "/b":
			fmt.Fprint(w, `<title>B</title>`)
		case "/deep":
			deep.Add(1)
			fmt.Fprint(w, `<title>Deep</title><a href="/deeper">z</a>`)
		case "/deeper":
			deeper.Add(1)
			fmt.Fprint(w, `<title>Deeper</title>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewCrawler(NewFetcher(time.Second, log.New(io.Discard, "", 0)))
	pages := c.Crawl(context.Background(), []string{srv.URL + "/", srv.URL + "/missing"}, 2)
	if len(pages) != 1 || pages[0].Title != "Home" || len(pages[0].Links) != 2 {
		t.Fatalf("pages = %+v", pages)
	}

	var pageA *Page
	for _, child := range pages[0].Links {
		switch child.Title {
		case "A":
			pageA = child
		case "B":
			if len(child.Links) != 0 {
				t.Fatalf("B links = %+v", child.Links)
			}
		default:
			t.Fatalf("unexpected child %+v", child)
		}
	}
	if pageA == nil || len(pageA.Links) != 1 || pageA.Links[0].Title != "Deep" || len(pageA.Links[0].Links) != 0 {
		t.Fatalf("A = %+v", pageA)
	}
	if home.Load() != 1 || deep.Load() != 1 || deeper.Load() != 0 {
		t.Fatalf("home=%d deep=%d deeper=%d", home.Load(), deep.Load(), deeper.Load())
	}
}

func TestCrawlCancel(t *testing.T) {
	started := make(chan struct{})
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-started
		cancel()
	}()

	pages := NewCrawler(NewFetcher(time.Second, log.New(io.Discard, "", 0))).Crawl(ctx, []string{srv.URL}, 2)
	if len(pages) != 0 {
		t.Fatalf("pages = %+v", pages)
	}
}
