package d1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

var (
	ErrUnexpectedStatusCode = errors.New("unexpected status code")
	ErrD1APIUnsuccessful    = errors.New("D1 API returned unsuccessful response")
)

// Response is the response from the D1 API.
type Response struct {
	Success bool `json:"success"`
	Result  []struct {
		Results []map[string]any `json:"results"`
	} `json:"result"`
}

// CloudflareAPI handles D1 API requests.
type CloudflareAPI struct {
	accountID string
	d1ID      string
	token     string
	client    *http.Client
}

// NewCloudflareAPI creates a new Cloudflare API client.
func NewCloudflareAPI(accountID, d1ID, token string) *CloudflareAPI {
	return &CloudflareAPI{
		accountID: accountID,
		d1ID:      d1ID,
		token:     token,
		client:    &http.Client{},
	}
}

// ExecuteSQL executes a SQL statement on D1 and returns the results.
func (c *CloudflareAPI) ExecuteSQL(ctx context.Context, sql string, params []any) ([]map[string]any, error) {
	url := fmt.Sprintf(
		"https://api.cloudflare.com/client/v4/accounts/%s/d1/database/%s/query",
		c.accountID,
		c.d1ID,
	)

	// Prepare request body
	body := map[string]any{
		"sql":    sql,
		"params": params,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %d: %s", ErrUnexpectedStatusCode, resp.StatusCode, string(body))
	}

	// Parse response
	var d1Resp Response
	if err := json.NewDecoder(resp.Body).Decode(&d1Resp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if !d1Resp.Success {
		return nil, ErrD1APIUnsuccessful
	}

	if len(d1Resp.Result) == 0 || len(d1Resp.Result[0].Results) == 0 {
		return []map[string]any{}, nil
	}

	return d1Resp.Result[0].Results, nil
}
