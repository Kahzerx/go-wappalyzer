package wappalyzer

import (
	"golang.org/x/net/html"
	"net/http"
	"net/url"
)

type WebPage struct {
	url     *url.URL
	html    *html.Node
	headers map[string][]string
	scripts []string
	meta    map[string]string
}

func NewWebpageFromResponse(response *http.Response) *WebPage {
	rUrl := response.Request.URL
	rHtml, err := html.Parse(response.Body)
	if err != nil {
		panic("Invalid html format")
	}
	headers := response.Header
	htmlParser := newDocumentParser(rHtml)
	scriptNodes := htmlParser.findAll("script", boolKeyArgs{"src": true})
	var scripts []string
	for _, s := range scriptNodes {
		for _, attr := range s.Attr {
			if attr.Key == "src" {
				scripts = append(scripts, attr.Val)
			}
		}
	}
	metaNodes := htmlParser.findAll("meta", boolKeyArgs{"name": true, "content": true})
	meta := make(map[string]string)
	for _, m := range metaNodes {
		name := ""
		content := ""
		for _, attr := range m.Attr {
			if attr.Key == "name" {
				name = attr.Val
			}
			if attr.Key == "content" {
				content = attr.Val
			}
		}
		meta[name] = content
	}
	return &WebPage{
		url:     rUrl,
		html:    rHtml,
		headers: headers,
		scripts: scripts,
		meta:    meta,
	}
}
