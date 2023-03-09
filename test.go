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
	//log.Println(fmt.Sprintf("%+v", webPage))
	wp := wappalyzer.NewWappalyzer(false)
	wp.AnalyzeWithVersions(webPage)
	//log.Println(fmt.Sprintf("%+v", wp))
}
