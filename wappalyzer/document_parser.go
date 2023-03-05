package wappalyzer

import (
	"golang.org/x/net/html"
)

type boolKeyArgs map[string]bool

type documentParser struct {
	html *html.Node
}

func newDocumentParser(doc *html.Node) *documentParser {
	return &documentParser{html: doc}
}

func (dp *documentParser) findAll(tagElement string, bk boolKeyArgs) []*html.Node {
	var requiredKeys []string
	for k, v := range bk {
		if v {
			requiredKeys = append(requiredKeys, k)
		}
	}
	var res []*html.Node
	node := func(n *html.Node) {
		mandatoryKeys := make(map[string]bool)
		for _, key := range requiredKeys {
			mandatoryKeys[key] = false
		}
		if n.Type == html.ElementNode && n.Data == tagElement {
			for _, a := range n.Attr {
				if dp.contains(requiredKeys, a.Key) {
					mandatoryKeys[a.Key] = true
				}
			}
			if dp.isValid(mandatoryKeys) {
				res = append(res, n)
			}
		}
	}
	dp.iterNodes(dp.html, node, nil)
	return res
}

func (dp *documentParser) isValid(check map[string]bool) bool {
	for _, found := range check {
		if !found {
			return false
		}
	}
	return true
}

func (dp *documentParser) contains(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

func (dp *documentParser) iterNodes(n *html.Node, pre, post func(n *html.Node)) {
	if pre != nil {
		pre(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		dp.iterNodes(c, pre, post)
	}
	if post != nil {
		post(n)
	}
}
