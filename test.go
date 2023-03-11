package main

import (
	"fmt"
	"log"
	"wappalyzer/wappalyzer"
)

func main() {
	webPage, err := wappalyzer.NewWebpage("https://google.com/")
	if err != nil {
		log.Println(err)
	}
	//log.Println(fmt.Sprintf("%+v", webPage))
	wp := wappalyzer.NewWappalyzer(false)
	log.Println(fmt.Sprintf("%+v", wp.AnalyzeWithVersions(webPage)))
	log.Println(fmt.Sprintf("%+v", wp.Analyze(webPage)))
}
