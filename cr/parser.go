package cr

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func Parse(pageUrl string, r io.Reader) (title string, links []string, err error) {
	base, err := url.Parse(pageUrl)
	if err != nil {
		return "", nil, err
	}
	doc, err := html.Parse(r)
	if err != nil {
		return "", nil, err
	}
	links = []string{}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && title == "" {
			title = strings.TrimSpace(nodeText(n))
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			if href := attr(n, "href"); href != "" {
				if abs, ok := resolve(base, href); ok {
					links = append(links, abs)
				}
			}
		}
	}
	f(doc)
	return title, links, nil
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}

func nodeText(n *html.Node) string {
	var b strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return b.String()

}

func resolve(base *url.URL, href string) (string, bool) {
	ref, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	abs := base.ResolveReference(ref)
	if abs.Scheme != "http" && abs.Scheme != "https" {
		return "", false
	}
	return abs.String(), true

}
