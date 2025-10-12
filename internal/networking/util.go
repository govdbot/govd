package networking

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/bytedance/sonic"
)

func (client *HTTPClient) Fetch(
	method string,
	url string,
	params *RequestParams,
) (*http.Response, error) {
	ctx := context.Background()

	return client.FetchWithContext(ctx, method, url, params)
}

func (client *HTTPClient) FetchWithContext(
	ctx context.Context,
	method string,
	url string,
	params *RequestParams,
) (*http.Response, error) {
	if params == nil {
		params = &RequestParams{}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, params.Body)
	if err != nil {
		return nil, err
	}
	// set client scoped headers and cookies
	// then override with request specific ones
	for k, v := range client.Headers {
		req.Header.Set(k, v)
	}
	for _, cookie := range client.Cookies {
		req.AddCookie(cookie)
	}
	for k, v := range params.Headers {
		req.Header.Set(k, v)
	}
	for _, cookie := range params.Cookies {
		req.AddCookie(cookie)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", generateChromeUA())
	}
	resp, err := client.Client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func generateChromeUA() string {
	// TODO: generate random UA
	return "Mozilla/5.0 (Linux; Android 10; SM-G960U) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.181 Mobile Safari/537.36"
}

func readRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading request body: %w", err)
	}

	req.Body.Close()
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return bodyBytes, nil
}

func parseEdgeProxyResponse(resp *http.Response, req *http.Request) (*http.Response, error) {
	var proxyResponse EdgeProxyResponse

	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	if err := decoder.Decode(&proxyResponse); err != nil {
		return nil, fmt.Errorf("error parsing proxy response: %w", err)
	}

	response := &http.Response{
		StatusCode: proxyResponse.StatusCode,
		Status:     strconv.Itoa(proxyResponse.StatusCode) + " " + http.StatusText(proxyResponse.StatusCode),
		Body:       io.NopCloser(bytes.NewBufferString(proxyResponse.Text)),
		Header:     make(http.Header),
		Request:    req,
	}

	parsedResponseURL, err := url.Parse(proxyResponse.URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing response URL: %w", err)
	}
	resp.Request.URL = parsedResponseURL

	for name, value := range proxyResponse.Headers {
		resp.Header.Set(name, value)
	}

	for _, cookie := range proxyResponse.Cookies {
		resp.Header.Add("Set-Cookie", cookie)
	}

	return response, nil
}
