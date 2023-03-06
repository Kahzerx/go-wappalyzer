package wappalyzer

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
)

type Wappalyzer struct {
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
	}
	return new(Wappalyzer)
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
