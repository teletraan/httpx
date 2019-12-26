package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// Client wrap http request.
type Client struct {
	BaseURL   *url.URL
	UserAgent string

	httpClient *http.Client
}

// New create a httpx Client.
func New(baseURL, userAgent string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if userAgent == "" {
		userAgent = "httpx"
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("create httpx client error: %w", err)
	}
	return &Client{BaseURL: u, UserAgent: userAgent, httpClient: httpClient}, nil
}

// Copy a new Client with http client.
func (c *Client) Copy(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{BaseURL: c.BaseURL, UserAgent: c.UserAgent, httpClient: httpClient}
}

// NewRequest creates an json API request. A relative URL can be provided in urlStr,
// in which case it is resolved relative to the BaseURL of the Client.
// Relative URLs should always be specified with a preceding slash.
func (c *Client) NewRequest(method, urlStr string, params map[string]string, body interface{}) (*http.Request, error) {
	if !strings.HasPrefix(urlStr, "/") {
		return nil, fmt.Errorf("httpx new request error: url must have a preceding slash, but %q does not", urlStr)
	}
	u, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("httpx new request error: %w", err)
	}
	if params != nil {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return nil, fmt.Errorf("httpx new request error: %w", err)
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, fmt.Errorf("httpx new request error: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	return req, nil
}

// NewMultiPartRequest creates an multi-part API request. A relative URL can be provided in urlStr,
// in which case it is resolved relative to the BaseURL of the Client.
// Relative URLs should always be specified with a preceding slash.
func (c *Client) NewMultiPartRequest(method, urlStr string, body io.Reader, contentType string) (*http.Request, error) {
	if body == nil {
		return nil, errors.New("httpx new multi part request error: expected not nil body")
	}
	if !strings.HasPrefix(urlStr, "/") {
		return nil, fmt.Errorf("httpx new multi part request error: url must have a preceding slash, but %q does not", urlStr)
	}
	u, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("httpx new multi part request error: %w", err)
	}

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("httpx new multi part request error: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	return req, nil
}

// Response wraps the standard http.Response.
type Response struct {
	*http.Response
	ErrMessage string
}

// HasError return response status code int [400, 599].
func (r *Response) HasError() bool {
	return 400 <= r.StatusCode && r.StatusCode <= 599
}

func (r *Response) Error() string {
	return fmt.Sprintf("%v %v: %d %v", r.Response.Request.Method, r.Response.Request.URL, r.Response.StatusCode, r.ErrMessage)
}

// Do sends an API request and returns the API response. The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred. If v implements the io.Writer
// interface, the raw response body will be written to v, without attempting to
// first decode it.
// The provided ctx must be non-nil. If it is canceled or times out,
// ctx.Err() will be returned.
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {
	if v == nil {
		return nil, errors.New("v must not nil")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("httpx do error: %w", ctx.Err())
		default:
		}

		return nil, fmt.Errorf("httpx do error: %w", err)
	}
	defer resp.Body.Close()
	response := &Response{Response: resp}

	if response.HasError() {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("httpx do error: %w", err)
		}
		response.ErrMessage = string(body)
		// return response as error
		return nil, response
	}

	if w, ok := v.(io.Writer); ok {
		io.Copy(w, resp.Body)
		return response, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil && !errors.Is(err, io.EOF) {
		return response, err
	}
	return response, nil
}
