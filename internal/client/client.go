// Package client is a minimal hand-written HTTP client for the policy.opteryx
// access-policy API (app/routes/v1/access.py). It intentionally covers only
// the endpoints this provider needs, not the full service surface.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Principal mirrors app/models/policy.py::Principal.
type Principal struct {
	Identity string `json:"identity"`
}

// PolicyInfo mirrors app/models/policy.py::PolicyInfo, one entry in the
// list-workspace-policies response.
type PolicyInfo struct {
	Identity string `json:"identity"`
	Role     string `json:"role"`
	Pattern  string `json:"pattern"`
	Policy   string `json:"policy"`
}

// PolicyDetail mirrors app/models/policy.py::PolicyDetail, the get-policy response.
type PolicyDetail struct {
	Principal Principal  `json:"principal"`
	Role      string     `json:"role"`
	Pattern   string     `json:"pattern"`
	CreatedAt *string    `json:"created_at"`
	UpdatedAt *string    `json:"updated_at"`
	UpdatedBy *Principal `json:"updated_by"`
}

// APIError carries the HTTP status and the service's own detail message, so
// callers can surface it verbatim -- e.g. the self-grant and pattern-authority
// 403s from create_policy/update_policy/delete_policy -- instead of a generic
// wrapper that hides why the call was rejected.
type APIError struct {
	StatusCode int
	Detail     string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("policy.opteryx: %d: %s", e.StatusCode, e.Detail)
}

// IsNotFound reports whether err is an APIError for a 404 response, so
// resource Read implementations can drop the resource from state.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling policy.opteryx: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 300 {
		detail := string(respBody)
		var errBody struct {
			Detail string `json:"detail"`
		}
		if json.Unmarshal(respBody, &errBody) == nil && errBody.Detail != "" {
			detail = errBody.Detail
		}
		return &APIError{StatusCode: resp.StatusCode, Detail: detail}
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decoding response body: %w", err)
		}
	}

	return nil
}

// ListPolicies calls GET /v1/access/workspace/{workspace}.
func (c *Client) ListPolicies(ctx context.Context, workspace string) ([]PolicyInfo, error) {
	var out struct {
		Principals []PolicyInfo `json:"principals"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/access/workspace/"+workspace, nil, &out); err != nil {
		return nil, err
	}
	return out.Principals, nil
}

// GetPolicy calls GET /v1/access/workspace/{workspace}/policies/{policyID}.
func (c *Client) GetPolicy(ctx context.Context, workspace, policyID string) (*PolicyDetail, error) {
	var out PolicyDetail
	path := fmt.Sprintf("/v1/access/workspace/%s/policies/%s", workspace, policyID)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreatePolicy calls POST /v1/access/workspace/{workspace}/policies and returns the new policy ID.
func (c *Client) CreatePolicy(ctx context.Context, workspace, principal, role, pattern string) (string, error) {
	reqBody := struct {
		Principal Principal `json:"principal"`
		Role      string    `json:"role"`
		Pattern   string    `json:"pattern"`
	}{
		Principal: Principal{Identity: principal},
		Role:      role,
		Pattern:   pattern,
	}
	var out struct {
		Policy  string `json:"policy"`
		Message string `json:"message"`
	}
	path := fmt.Sprintf("/v1/access/workspace/%s/policies", workspace)
	if err := c.do(ctx, http.MethodPost, path, reqBody, &out); err != nil {
		return "", err
	}
	return out.Policy, nil
}

// UpdatePolicy calls PUT /v1/access/workspace/{workspace}/policies/{policyID}.
// The API has no way to change a policy's principal -- UpdatePolicyRequest
// only carries role and pattern -- so this is the entire mutable surface.
func (c *Client) UpdatePolicy(ctx context.Context, workspace, policyID, role, pattern string) error {
	reqBody := struct {
		Role    string `json:"role"`
		Pattern string `json:"pattern"`
	}{
		Role:    role,
		Pattern: pattern,
	}
	path := fmt.Sprintf("/v1/access/workspace/%s/policies/%s", workspace, policyID)
	return c.do(ctx, http.MethodPut, path, reqBody, nil)
}

// DeletePolicy calls DELETE /v1/access/workspace/{workspace}/policies/{policyID}.
func (c *Client) DeletePolicy(ctx context.Context, workspace, policyID string) error {
	path := fmt.Sprintf("/v1/access/workspace/%s/policies/%s", workspace, policyID)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}
