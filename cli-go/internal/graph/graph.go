// Package graph is a minimal read-only client for the WhatsApp Business Management
// API on Meta's Graph API. It sends GET requests only and makes no retries.
package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultVersion is the Graph API version used when none is configured.
const DefaultVersion = "v25.0"

// Client reads one WhatsApp Business Account (WABA) with one access token.
type Client struct {
	Base    string // https://graph.facebook.com
	Version string
	Token   string
	HTTP    *http.Client
}

// Error is Meta's error envelope, kept whole so callers can see code and trace id.
type Error struct {
	Status  int    `json:"-"`
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
	Subcode int    `json:"error_subcode"`
	TraceID string `json:"fbtrace_id"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("graph API %d (code %d, subcode %d): %s [trace %s]", e.Status, e.Code, e.Subcode, e.Message, e.TraceID)
}

// Get requests path (for example "/123/phone_numbers") with query and returns the raw JSON body.
func (c *Client) Get(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("no access token: set WHATSAPP_ACCESS_TOKEN (a system user token with whatsapp_business_management)")
	}
	base := strings.TrimRight(c.Base, "/")
	if base == "" {
		base = "https://graph.facebook.com"
	}
	version := c.Version
	if version == "" {
		version = DefaultVersion
	}
	target := base + "/" + version + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	// The token travels in a header, never in the URL, so it cannot leak into logs or errors.
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var envelope struct {
			Error *Error `json:"error"`
		}
		if json.Unmarshal(body, &envelope) == nil && envelope.Error != nil {
			envelope.Error.Status = resp.StatusCode
			return nil, envelope.Error
		}
		return nil, &Error{Status: resp.StatusCode, Message: strings.TrimSpace(string(body))}
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("graph API returned a non-JSON body (%d bytes)", len(body))
	}
	return body, nil
}

// Expression builds a Graph field expression such as
// pricing_analytics.start(1).end(2).granularity(DAILY).dimensions(["COUNTRY"]).
// Filters keep the order given; list values are JSON arrays of strings.
func Expression(field string, filters []Filter) string {
	var b strings.Builder
	b.WriteString(field)
	for _, f := range filters {
		if f.List != nil {
			encoded, _ := json.Marshal(f.List)
			fmt.Fprintf(&b, ".%s(%s)", f.Name, encoded)
		} else {
			fmt.Fprintf(&b, ".%s(%s)", f.Name, f.Value)
		}
	}
	return b.String()
}

// Filter is one dotted filter in a field expression.
type Filter struct {
	Name  string
	Value string
	List  []string
}
