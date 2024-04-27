package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	client  *http.Client
	timeout = time.Second * 10
)

func init() {
	client = &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(network, addr, timeout)
				if err != nil {
					return nil, err
				}
				err = conn.SetDeadline(time.Now().Add(timeout))
				if err != nil {
					return nil, err
				}
				return conn, nil
			},
			ResponseHeaderTimeout: timeout,
		},
	}
}

func PostForm(ctx context.Context, urlStr string, formList map[string]string) (response []byte, err error) {
	var temp []string
	for key, value := range formList {
		temp = append(temp, key+"="+url.QueryEscape(value))
	}
	implodedStr := strings.Join(temp, "&")

	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, strings.NewReader(implodedStr))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	body, err := getResponse(req)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func PostJson(ctx context.Context, urlStr string, formList map[string]any) (response []byte, err error) {
	jsonBytes, errJson := json.Marshal(formList)
	if errJson != nil {
		return nil, errJson
	}
	fmt.Println(urlStr)
	fmt.Println(string(jsonBytes))

	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	body, err := getResponse(req)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func PostData(ctx context.Context, urlStr string, queryList map[string]string, data []byte) (response []byte, err error) {
	var temp = make([]string, 0)
	for key, value := range queryList {
		stringQuery := key + "=" + url.QueryEscape(value)
		temp = append(temp, stringQuery)
	}

	queryVar := strings.Join(temp, "&")
	fullQuery := urlStr + "?" + queryVar

	req, err := http.NewRequestWithContext(ctx, "POST", fullQuery, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	body, err := getResponse(req)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func Get(ctx context.Context, addr string, queryList map[string]string, headers map[string]string) (response []byte, err error) {
	var temp = make([]string, 0, len(queryList))
	for key, value := range queryList {
		stringQuery := key + "=" + url.QueryEscape(value)
		temp = append(temp, stringQuery)
	}

	var queryVar = strings.Join(temp, "&")
	fullQuery := addr + "?" + queryVar

	req, reqErr := http.NewRequestWithContext(ctx, "GET", fullQuery, nil)
	if reqErr != nil {
		return nil, reqErr
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	body, err := getResponse(req)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func getResponse(req *http.Request) ([]byte, error) {
	resp, err := client.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
