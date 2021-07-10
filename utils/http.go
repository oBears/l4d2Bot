package utils

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func Fetch(uri string, method string, contentType string, data map[string]string) (string, error) {
	client := &http.Client{}
	var req *http.Request
	switch contentType {
	case "application/json":
		bytesData, err := json.Marshal(data)
		if err != nil {
			return "", err
		}
		reader := bytes.NewReader(bytesData)
		req, _ = http.NewRequest(method, uri, reader)
		req.Header.Add("Content-Type", "application/json")

	case "application/x-www-form-urlencoded":
		postData := url.Values{}
		for key, value := range data {
			postData.Set(key, value)
		}
		req, _ = http.NewRequest(method, uri, strings.NewReader(postData.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	default:
		req, _ = http.NewRequest(method, uri, nil)

	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.90 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
