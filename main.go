package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"crawlerCli/cr"
)

func main() {
	urlsFlag := flag.String("urls", "", "стартовые URL через запятую")
	depth := flag.Int("depth", 1, "глубина обхода ссылок")
	timeout := flag.Duration("timeout", time.Minute, "общий таймаут")
	requestTimeout := flag.Duration("request-timeout", 10*time.Second, "таймаут одного запроса")
	output := flag.String("output", "result.json", "файл результата JSON")
	logPath := flag.String("log", "crawler.log", "файл ошибок и HTTP-статусов")
	flag.Parse()

	urls := splitURLs(*urlsFlag)
	if len(urls) == 0 || *depth < 0 || *timeout <= 0 || *requestTimeout <= 0 {
		fmt.Fprintln(os.Stderr, "нужны --urls и положительные --timeout, --request-timeout; --depth >= 0")
		flag.Usage()
		os.Exit(2)
	}

	logFile, err := os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "log: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	fetcher := cr.NewFetcher(*requestTimeout, log.New(logFile, "", log.LstdFlags))
	pages := cr.NewCrawler(fetcher).Crawl(ctx, urls, *depth)
	if err := writeJSON(*output, pages); err != nil {
		fmt.Fprintf(os.Stderr, "output: %v\n", err)
		os.Exit(1)
	}
}

func splitURLs(raw string) []string {
	parts := strings.Split(raw, ",")
	urls := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			urls = append(urls, part)
		}
	}
	return urls
}

func writeJSON(path string, pages []*cr.Page) error {
	if pages == nil {
		pages = []*cr.Page{}
	}
	data, err := json.MarshalIndent(pages, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
