package zenmoney

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://api.zenmoney.ru"

// Client is the ZenMoney API client.
type Client struct {
	token      string
	httpClient *http.Client
}

// NewClient creates a client with the given access token.
func NewClient(accessToken string) *Client {
	return &Client{
		token: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetHTTPClient overrides the default HTTP client.
func (c *Client) SetHTTPClient(hc *http.Client) {
	c.httpClient = hc
}

func (c *Client) do(method, path string, body any) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		r = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s %s: %w", method, path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(errBody),
		}
	}

	return resp, nil
}

// APIError represents a non-2xx response from the ZenMoney API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("zenmoney: HTTP %d: %s", e.StatusCode, e.Body)
}

// Diff performs a bidirectional sync with the ZenMoney server.
// Pass serverTimestamp=0 for initial full sync.
func (c *Client) Diff(req *DiffRequest) (*DiffResponse, error) {
	if req.CurrentClientTimestamp == 0 {
		req.CurrentClientTimestamp = time.Now().Unix()
	}

	resp, err := c.do(http.MethodPost, "/v8/diff/", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var diff DiffResponse
	if err := json.NewDecoder(resp.Body).Decode(&diff); err != nil {
		return nil, fmt.Errorf("decode diff response: %w", err)
	}
	return &diff, nil
}

// FetchAll performs an initial full sync, returning all data from the server.
func (c *Client) FetchAll() (*DiffResponse, error) {
	return c.Diff(&DiffRequest{
		ServerTimestamp: 0,
	})
}

// Suggest requests category/merchant suggestions for a transaction.
func (c *Client) Suggest(req *SuggestRequest) (*SuggestResponse, error) {
	resp, err := c.do(http.MethodPost, "/v8/suggest/", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var sug SuggestResponse
	if err := json.NewDecoder(resp.Body).Decode(&sug); err != nil {
		return nil, fmt.Errorf("decode suggest response: %w", err)
	}
	return &sug, nil
}

// SuggestBatch requests suggestions for multiple transactions at once.
func (c *Client) SuggestBatch(reqs []SuggestRequest) ([]SuggestResponse, error) {
	resp, err := c.do(http.MethodPost, "/v8/suggest/", reqs)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var sugs []SuggestResponse
	if err := json.NewDecoder(resp.Body).Decode(&sugs); err != nil {
		return nil, fmt.Errorf("decode suggest batch response: %w", err)
	}
	return sugs, nil
}

// Push sends local changes to the server without requesting updates.
// Convenience wrapper around Diff that only sends mutations.
func (c *Client) Push(serverTimestamp int64, opts ...PushOption) (*DiffResponse, error) {
	req := &DiffRequest{
		ServerTimestamp: serverTimestamp,
	}
	for _, o := range opts {
		o(req)
	}
	return c.Diff(req)
}

// PushOption configures what data to send in a Push call.
type PushOption func(*DiffRequest)

func WithAccounts(a []Account) PushOption         { return func(r *DiffRequest) { r.Account = a } }
func WithTags(t []Tag) PushOption                 { return func(r *DiffRequest) { r.Tag = t } }
func WithMerchants(m []Merchant) PushOption        { return func(r *DiffRequest) { r.Merchant = m } }
func WithTransactions(t []Transaction) PushOption  { return func(r *DiffRequest) { r.Transaction = t } }
func WithBudgets(b []Budget) PushOption            { return func(r *DiffRequest) { r.Budget = b } }
func WithReminders(r []Reminder) PushOption        { return func(req *DiffRequest) { req.Reminder = r } }
func WithDeletions(d []Deletion) PushOption         { return func(r *DiffRequest) { r.Deletion = d } }
