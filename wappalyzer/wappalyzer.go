package wappalyzer

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
)

type Wappalyzer struct {
	technologies     map[string]interface{}
	confidenceRegexp *regexp.Regexp
}

func NewWappalyzer(update bool) *Wappalyzer {
	techDir := "technologies"
	if update {
		downloadTechs(techDir)
	}
	if !ensureDirIsValid(techDir) {
		return nil
	}
	technologies := setupTechs(techDir)
	for name, tech := range technologies {
		t, ok := tech.(map[string]interface{})
		if !ok {
			log.Println(fmt.Sprintf("Not a valid format found for %s, skipping...", name))
			continue
		}
		prepareTech(t)
		//log.Println(fmt.Sprintf("%+v", tech))
	}
	wappalyzer := new(Wappalyzer)
	wappalyzer.technologies = technologies
	compile, _ := regexp.Compile("(.+)\\;confidence:(\\d+)")
	wappalyzer.confidenceRegexp = compile
	return wappalyzer
}

func (wp *Wappalyzer) AnalyzeWithVersions(page *WebPage) {
	wp.Analyze(page)
}

func (wp *Wappalyzer) Analyze(page *WebPage) {
	techsfound := make(map[string]bool)
	_ = techsfound
	for techName, tech := range wp.technologies {
		if wp.hasTech(tech.(map[string]interface{}), page) {
			techsfound[techName] = true
		}
	}
	//log.Println(fmt.Sprintf("%+v", page.scripts))
}

func (wp *Wappalyzer) hasTech(tech map[string]interface{}, page *WebPage) bool {
	//log.Println("====")
	//log.Println(tech)
	found := false
	//log.Println(fmt.Sprintf("%+v", tech))
	for _, pattern := range tech["url"].([]map[string]interface{}) {
		if p := pattern["regex"]; p != nil && p.(*regexp.Regexp).MatchString(page.url) {
			log.Println("aaa")
		}
	}
	for name, pattern := range tech["headers"].(map[string]interface{}) {
		if headerContent := page.headers[name]; headerContent != nil && headerContent[0] != "" {
			if p := pattern.(map[string]interface{})["regex"]; p != nil && p.(*regexp.Regexp).MatchString(headerContent[0]) {
				log.Println("bbb")
				found = true
			}
		}
	}
	for _, pattern := range tech["scriptSrc"].([]map[string]interface{}) {
		for _, script := range page.scripts {
			if p := pattern["regex"]; p != nil && p.(*regexp.Regexp).MatchString(script) {
				log.Println("ccc")
				found = true
			}
		}
	}
	for name, pattern := range tech["meta"].(map[string]interface{}) {
		if metaContent := page.meta[name]; metaContent != "" {
			if p := pattern.(map[string]interface{})["regex"]; p != nil && p.(*regexp.Regexp).MatchString(metaContent) {
				log.Println("ddd")
				found = true
			}
		}
	}
	for _, pattern := range tech["html"].([]map[string]interface{}) {
		if p := pattern["regex"]; p != nil && p.(*regexp.Regexp).MatchString(page.rawHtml) {
			log.Println("eee")
			found = true
		}
	}
	return found
}

func prepareTech(tech map[string]interface{}) {
	// Ensure this keys exist and are slices
	for _, k := range []string{"url", "html", "scriptSrc", "implies"} {
		val := tech[k]
		if val == nil {
			tech[k] = []string{}
			continue
		}

		if reflect.TypeOf(tech[k]).Kind() != reflect.Slice {
			strVal, ok := tech[k].(string)
			if !ok {
				log.Println(fmt.Sprintf("%s has an unknown type, skipping... please report!", tech[k]))
				continue
			}
			tech[k] = []string{strVal}
		} else {
			var patterns []string
			for _, pat := range tech[k].([]interface{}) {
				patterns = append(patterns, pat.(string))
			}
			tech[k] = patterns
		}
	}

	// Ensure this keys exist and are map
	for _, k := range []string{"headers", "meta"} {
		val := tech[k]
		if val == nil {
			tech[k] = make(map[string]interface{})
			continue
		}
	}
	if reflect.TypeOf(tech["meta"]).Kind() != reflect.Map {
		gen := make(map[string]interface{})
		gen["generator"] = tech["meta"]
		tech["meta"] = gen
	}
	meta := tech["meta"].(map[string]interface{})
	if meta["generator"] != nil && reflect.TypeOf(meta["generator"]).Kind() == reflect.Slice {
		tech["meta"].(map[string]interface{})["generator"] = meta["generator"].([]interface{})[0]
	}

	for _, k := range []string{"headers", "meta"} {
		val := tech[k].(map[string]interface{})
		for key, value := range val {
			tech[strings.ToLower(key)] = value
		}
	}

	for _, k := range []string{"url", "html", "scriptSrc"} {
		var patterns []map[string]interface{}
		for _, t := range tech[k].([]string) {
			patterns = append(patterns, preparePattern(t))
		}
		tech[k] = patterns
	}
	for _, k := range []string{"headers", "meta"} {
		val := tech[k].(map[string]interface{})
		for key, pattern := range val {
			tech[k].(map[string]interface{})[key] = preparePattern(pattern.(string))
		}
	}
}

func preparePattern(pattern string) map[string]interface{} {
	attrs := make(map[string]interface{})
	patterns := strings.Split(pattern, "\\;")
	for i, expr := range patterns {
		if i == 0 {
			compile, err := regexp.Compile(fmt.Sprintf("(?i)%s", expr))
			if err != nil {
				log.Println(fmt.Sprintf("Unable to compile %s, skipping...", expr))
				continue
			}
			attrs["string"] = expr
			attrs["regex"] = compile
		} else {
			attr := strings.SplitN(expr, ":", 2)
			attrs[attr[0]] = attr[1]
		}
	}
	return attrs
}

func ensureDirIsValid(techDir string) bool {
	_, err := os.Stat(techDir)
	if err != nil {
		return false
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func setupTechs(techDir string) map[string]interface{} {
	technologies := make(map[string]interface{})
	dir, _ := os.ReadDir(techDir)
	for _, file := range dir {
		if strings.HasSuffix(file.Name(), ".json") {
			var content map[string]interface{}
			f, err := os.ReadFile(fmt.Sprintf("%s/%s", techDir, file.Name()))
			if err != nil {
				continue
			}
			err = json.Unmarshal(f, &content)
			if err != nil {
				continue
			}
			for k, v := range content {
				technologies[k] = v
			}
		}
	}
	return technologies
}

func downloadTechs(techDir string) {
	asciiLowercase := "abcdefghijklmnopqrstuvwxyz_"
	_ = os.Mkdir(techDir, os.ModePerm)
	for _, letter := range strings.Split(asciiLowercase, "") {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://raw.githubusercontent.com/wappalyzer/wappalyzer/master/src/technologies/%s.json", letter), nil)
		if err != nil {
			return
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}
		bytes, err := io.ReadAll(response.Body)
		if err != nil {
			return
		}
		_ = response.Body.Close()
		_ = os.WriteFile(fmt.Sprintf("%s/%s.json", techDir, letter), bytes, os.ModePerm)
	}
}
