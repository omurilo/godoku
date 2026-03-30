package generator

import (
	"fmt"
	"strings"

	"github.com/omurilo/godoku/internal/openapi"
)

type APIExampleLanguage string

const (
	LangCurl   APIExampleLanguage = "curl"
	LangGo     APIExampleLanguage = "go"
	LangPython APIExampleLanguage = "python"
	LangJS     APIExampleLanguage = "js"
)

func APIExample(lang APIExampleLanguage, server string, endpoint openapi.Endpoint, contentType string) string {
	switch lang {
	case LangCurl:
		return CurlExample(server, endpoint, contentType)
	case LangGo:
		return GoExample(server, endpoint, contentType)
	case LangPython:
		return PythonExample(server, endpoint, contentType)
	case LangJS:
		return JSExample(server, endpoint, contentType)
	default:
		return "// Example not available for language: " + string(lang)
	}
}

func CurlExample(server string, endpoint openapi.Endpoint, contentType string) string {

	url := openapi.BuildExampleURL(server, endpoint.Path, endpoint.Parameters)
	headers := openapi.GetRequiredHeaders(endpoint.Parameters)
	headers["Content-Type"] = contentType
	var headerLines []string
	for k, v := range headers {
		headerLines = append(headerLines, fmt.Sprintf("  -H '%s: %s' \\", k, v))
	}
	body := ""
	if endpoint.RequestBody != nil {
		for ct, media := range endpoint.RequestBody.Content {
			if ct == contentType || contentType == "" {
				example := openapi.GenerateExampleObject(media.Schema)
				example = strings.TrimSpace(example)
				example = strings.ReplaceAll(example, "\r", "")
				example = indentString(example, "  ")

				if strings.HasPrefix(example, "[") {
					body = fmt.Sprintf("  --data '%s' \\", example)
				} else {
					body = fmt.Sprintf("  --data '%s' \\", strings.ReplaceAll(example, "'", "\\'"))
				}
				break
			}
		}
	}
	lines := []string{fmt.Sprintf("curl -X %s \\", endpoint.Method)}
	lines = append(lines, headerLines...)
	if body != "" {
		lines = append(lines, body)
	}
	lines = append(lines, fmt.Sprintf("  '%s'", url))
	for i := range lines {
		if i == len(lines)-1 {
			lines[i] = strings.TrimSuffix(lines[i], " \\")
		}
	}
	return strings.Join(lines, "\n")
}

func GoExample(server string, endpoint openapi.Endpoint, contentType string) string {
	url := openapi.BuildExampleURL(server, endpoint.Path, endpoint.Parameters)
	headers := openapi.GetRequiredHeaders(endpoint.Parameters)
	headers["Content-Type"] = contentType
	body := "nil"
	if endpoint.RequestBody != nil {
		for ct, media := range endpoint.RequestBody.Content {
			if ct == contentType || contentType == "" {
				example := openapi.GenerateExampleObject(media.Schema)

				example = strings.TrimSpace(example)
				example = strings.ReplaceAll(example, "\r", "")
				example = indentString(example, "\t")

				body = fmt.Sprintf("strings.NewReader(`%s`)", example)
				break
			}
		}
	}
	var headerLines []string
	for k, v := range headers {
		headerLines = append(headerLines, fmt.Sprintf("\treq.Header.Add(\"%s\", \"%s\")", k, v))
	}
	return fmt.Sprintf(`package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	url := "%s"

	payload := %s

	req, err := http.NewRequest("%s", url, payload)
	if err != nil {
		panic(err)
	}

%s

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}
`, url, body, endpoint.Method, strings.Join(headerLines, "\n"))
}

func PythonExample(server string, endpoint openapi.Endpoint, contentType string) string {
	url := openapi.BuildExampleURL(server, endpoint.Path, endpoint.Parameters)
	headers := openapi.GetRequiredHeaders(endpoint.Parameters)
	headers["Content-Type"] = contentType
	var headerLines []string
	for k, v := range headers {
		headerLines = append(headerLines, fmt.Sprintf("    '%s': '%s'", k, v))
	}
	headersStr := "{}"
	if len(headerLines) > 0 {
		headersStr = fmt.Sprintf("{\n%s\n}", strings.Join(headerLines, ",\n"))
	}
	body := "None"
	if endpoint.RequestBody != nil {
		for ct, media := range endpoint.RequestBody.Content {
			if ct == contentType || contentType == "" {
				example := openapi.GenerateExampleObject(media.Schema)
				example = strings.TrimSpace(example)
				example = strings.ReplaceAll(example, "\r", "")

				if strings.HasPrefix(example, "[") {
					body = example
				} else {
					body = fmt.Sprintf("'''%s'''", example)
				}
				break
			}
		}
	}
	dataLine := ""
	if body != "None" {
		dataLine = "    data=body,\n"
	}
	return fmt.Sprintf(`import requests

url = '%s'
headers = %s
body = %s

response = requests.request(
    method='%s',
    url=url,
    headers=headers,
%s)

print(response.status_code)
print(response.json())`, url, headersStr, body, endpoint.Method, dataLine)
}

func JSExample(server string, endpoint openapi.Endpoint, contentType string) string {
	url := openapi.BuildExampleURL(server, endpoint.Path, endpoint.Parameters)
	headers := openapi.GetRequiredHeaders(endpoint.Parameters)
	headers["Content-Type"] = contentType
	var headerLines []string
	for k, v := range headers {
		headerLines = append(headerLines, fmt.Sprintf("    '%s': '%s'", k, v))
	}
	headersStr := "{}"
	if len(headerLines) > 0 {
		headersStr = fmt.Sprintf("{\n%s\n}", strings.Join(headerLines, ",\n"))
		headersStr = indentString(headersStr, "\t")
	}
	body := ""
	if endpoint.RequestBody != nil {
		for ct, media := range endpoint.RequestBody.Content {
			if ct == contentType || contentType == "" {
				example := openapi.GenerateExampleObject(media.Schema)
				example = strings.TrimSpace(example)
				example = strings.ReplaceAll(example, "\r", "")
				example = indentString(example, "\t")

				if strings.HasPrefix(example, "[") {
					body = fmt.Sprintf(",\n\tbody: %s", example)
				} else {
					body = fmt.Sprintf(",\n\tbody: `%s`", example)
				}
				break
			}
		}
	}
	return fmt.Sprintf(`async function callApi() {
    const response = await fetch('%s', {
        method: '%s',
        headers: %s%s
    });

    if (!response.ok) {
        throw new Error('Request failed: ' + response.status);
    }

    const data = await response.json();
    console.log(data);
}

callApi().catch(console.error);`, url, endpoint.Method, headersStr, body)
}

func indentString(str string, indent string) string {
	lines := strings.Split(str, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = indent + lines[i]
	}
	return strings.Join(lines, "\n")
}
