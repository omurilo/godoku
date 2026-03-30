package openapi

import (
	"fmt"
	"net/url"
	"strings"
)

func BuildExampleURL(server, path string, params []Parameter) string {
	urlPath := path
	query := url.Values{}
	for _, p := range params {
		ex := GenerateExampleValue(p.Schema, p.Name)
		exStr := fmt.Sprintf("%v", ex)
		switch p.In {
		case "path":
			urlPath = strings.ReplaceAll(urlPath, "{"+p.Name+"}", exStr)
		case "query":
			query.Set(p.Name, exStr)
		}
	}
	finalURL := strings.TrimRight(server, "/") + urlPath
	if len(query) > 0 {
		finalURL += "?" + query.Encode()
	}
	return finalURL
}

func GetRequiredHeaders(params []Parameter) map[string]string {
	headers := map[string]string{}
	for _, p := range params {
		if p.In == "header" && p.Required {
			headers[p.Name] = fmt.Sprintf("%v", GenerateExampleValue(p.Schema, p.Name))
		}
	}
	return headers
}
