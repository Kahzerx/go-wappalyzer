package main

import (
	"log"
	"wappalyzer/wappalyzer"
)

func main() {
	webPage, err := wappalyzer.NewWebpage("https://github.com/")
	if err != nil {
		log.Println(err)
	}
	//_ = webPage
	log.Println(webPage)
}
