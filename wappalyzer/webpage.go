package wappalyzer

import (
	"fmt"
	"golang.org/x/net/html"
	"net/http"
)

type WebPage struct {
	url     string
	html    *html.Node
	headers map[string][]string
	scripts []string
	meta    map[string]string
}

func NewWebpage(url string) (*WebPage, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create a valid request")
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to do the request")
	}
	return NewWebpageFromResponse(response)
}

func NewWebpageFromResponse(response *http.Response) (*WebPage, error) {
	rUrl := response.Request.URL
	rHtml, err := html.Parse(response.Body)
	if err != nil {
		return nil, fmt.Errorf("invalid html format")
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
		url:     rUrl.String(),
		html:    rHtml,
		headers: headers,
		scripts: scripts,
		meta:    meta,
	}, nil
}
