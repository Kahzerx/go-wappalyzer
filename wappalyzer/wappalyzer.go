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
	"strconv"
	"strings"
)

type Technologies struct {
	TechnologyName string
}

type TechnologiesWithVersion struct {
	TechnologyName string
	Versions       []string
}

type Wappalyzer struct {
	technologies     map[string]interface{}
	confidenceRegexp *regexp.Regexp
	detectedTech     map[string]interface{}
}

func NewWappalyzer(update bool) *Wappalyzer {
	techDir := "technologies"
	return NewWappalyzerWithDir(update, techDir)
}

func NewWappalyzerWithDir(update bool, jsonDir string) *Wappalyzer {
	if update {
		err := downloadTechs(jsonDir)
		if err != nil {
			return nil
		}
	}
	if !ensureDirIsValid(jsonDir) {
		return nil
	}
	technologies := setupTechs(jsonDir)
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
	compile, _ := regexp.Compile("(.+);confidence:(\\d+)")
	wappalyzer.confidenceRegexp = compile
	wappalyzer.detectedTech = make(map[string]interface{})
	return wappalyzer
}

func (wp *Wappalyzer) AnalyzeWithVersions(page *WebPage) []TechnologiesWithVersion {
	techsFound := wp.Analyze(page)
	var res []TechnologiesWithVersion
	for _, t := range techsFound {
		detectedTech := wp.detectedTech[t.TechnologyName]
		var versions []string
		if detectedTech != nil {
			if v := detectedTech.(map[string]interface{})["versions"]; v != nil {
				versions = v.([]string)
			}
		}
		res = append(res, TechnologiesWithVersion{
			TechnologyName: t.TechnologyName,
			Versions:       versions,
		})
	}
	return res
}

func (wp *Wappalyzer) Analyze(page *WebPage) []Technologies {
	wp.detectedTech = make(map[string]interface{})
	techsfound := make(map[string]bool)
	for techName, tech := range wp.technologies {
		if wp.hasTech(techName, tech.(map[string]interface{}), page) {
			techsfound[techName] = true
		}
	}
	wp.getImplied(techsfound)
	var res []Technologies
	for tech := range techsfound {
		res = append(res, Technologies{TechnologyName: tech})
	}
	return res
	//log.Println(fmt.Sprintf("%+v", wp.technologies))
}

func (wp *Wappalyzer) getImplied(techsFound map[string]bool) {
	newFound := false
	for tech := range techsFound {
		for _, impl := range wp.technologies[tech].(map[string]interface{})["implies"].([]string) {
			exists := techsFound[impl]
			if !exists {
				techsFound[impl] = true
				newFound = true
			}
		}
	}
	if newFound {
		wp.getImplied(techsFound)
	}
}

func (wp *Wappalyzer) hasTech(techName string, tech map[string]interface{}, page *WebPage) bool {
	//log.Println("====")
	//log.Println(tech)
	found := false
	//log.Println(fmt.Sprintf("%+v", tech))

	for _, pattern := range tech["url"].([]map[string]interface{}) {
		if p := pattern["regex"]; p != nil && p.(*regexp.Regexp).MatchString(page.url) {
			wp.setDetectedTech(techName, "url", pattern, page.url, "")
		}
	}
	for name, pattern := range tech["headers"].(map[string]interface{}) {
		if headerContent := page.headers[name]; headerContent != nil && headerContent[0] != "" {
			if p := pattern.(map[string]interface{})["regex"]; p != nil && p.(*regexp.Regexp).MatchString(headerContent[0]) {
				wp.setDetectedTech(techName, "headers", pattern.(map[string]interface{}), headerContent[0], name)
				found = true
			}
		}
	}
	for _, pattern := range tech["scriptSrc"].([]map[string]interface{}) {
		for _, script := range page.scripts {
			if p := pattern["regex"]; p != nil && p.(*regexp.Regexp).MatchString(script) {
				wp.setDetectedTech(techName, "scriptSrc", pattern, script, "")
				found = true
			}
		}
	}
	for name, pattern := range tech["meta"].(map[string]interface{}) {
		if metaContent := page.meta[name]; metaContent != "" {
			if p := pattern.(map[string]interface{})["regex"]; p != nil && p.(*regexp.Regexp).MatchString(metaContent) {
				wp.setDetectedTech(techName, "meta", pattern.(map[string]interface{}), metaContent, name)
				found = true
			}
		}
	}
	for _, pattern := range tech["html"].([]map[string]interface{}) {
		if p := pattern["regex"]; p != nil && p.(*regexp.Regexp).MatchString(page.rawHtml) {
			wp.setDetectedTech(techName, "html", pattern, page.rawHtml, "")
			found = true
		}
	}

	if found {
		if confidence := wp.detectedTech[techName].(map[string]interface{})["confidence"]; confidence != nil {
			total := 0
			for _, v := range confidence.(map[string]interface{}) {
				if reflect.TypeOf(v).Kind() == reflect.String {
					v, _ = strconv.Atoi(v.(string))
				}
				total += v.(int)
			}
			wp.detectedTech[techName].(map[string]interface{})["confidenceTotal"] = total
		}
	}

	return found
}

func (wp *Wappalyzer) setDetectedTech(techName string, techType string, pattern map[string]interface{}, val string, key string) {
	detected := wp.detectedTech[techName]
	if detected == nil {
		wp.detectedTech[techName] = make(map[string]interface{})
		detected = wp.detectedTech[techName]
	}
	if key != "" {
		key += " "
	}
	if confidence := detected.(map[string]interface{})["confidence"]; confidence == nil {
		detected.(map[string]interface{})["confidence"] = make(map[string]interface{})
	}
	if confidence := pattern["confidence"]; confidence == nil {
		pattern["confidence"] = 100
	}
	detected.(map[string]interface{})["confidence"].(map[string]interface{})[fmt.Sprintf("%s %s%s", techType, key, pattern["string"])] = pattern["confidence"]

	version := pattern["version"]
	if version != nil {
		matches := pattern["regex"].(*regexp.Regexp).FindStringSubmatch(val)
		if matches != nil {
			version = matches[1]
		}
		if version != "" {
			versions := detected.(map[string]interface{})["versions"]
			if versions == nil {
				detected.(map[string]interface{})["versions"] = []string{version.(string)}
			} else {
				exists := false
				for _, v := range versions.([]string) {
					if v == version {
						exists = true
					}
				}
				if !exists {
					versions = append(versions.([]string), version.(string))
				}
			}
		}
	}
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
				// log.Println(fmt.Sprintf("Unable to compile %s, skipping...", expr))
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

func downloadTechs(techDir string) error {
	asciiLowercase := "abcdefghijklmnopqrstuvwxyz_"
	_ = os.Mkdir(techDir, os.ModePerm)
	for _, letter := range strings.Split(asciiLowercase, "") {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://raw.githubusercontent.com/wappalyzer/wappalyzer/master/src/technologies/%s.json", letter), nil)
		if err != nil {
			return fmt.Errorf("request error")
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("request error")
		}
		if response.StatusCode != 200 {
			return fmt.Errorf("got status code %d", response.StatusCode)
		}
		bytes, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("failed to read request body")
		}
		_ = response.Body.Close()
		_ = os.WriteFile(fmt.Sprintf("%s/%s.json", techDir, letter), bytes, os.ModePerm)
	}
	return nil
}
