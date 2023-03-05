package main

import (
	"log"
	"net/http"
	"wappalyzer/wappalyzer"
)

func main() {
	req, err := http.NewRequest(http.MethodGet, "https://github.com/", nil)
	if err != nil {
		log.Println("Unable to create a valid request")
		return
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Unable to do the request")
		return
	}
	webPage := wappalyzer.NewWebpageFromResponse(response)
	//_ = webPage
	log.Println(webPage)
}
