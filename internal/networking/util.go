package networking

import (
	"context"
	"net/http"
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
		req.Header.Set("User-Agent", GenerateChromeUA())
	}
	resp, err := client.Client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func GenerateChromeUA() string {
	// TODO: generate random UA
	return "Mozilla/5.0 (Linux; Android 10; SM-G960U) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.181 Mobile Safari/537.36"
}
