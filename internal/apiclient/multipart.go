package apiclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

func newMultipartUploadBody(field, filename string, src io.Reader) (io.ReadCloser, string) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		part, err := mw.CreateFormFile(field, filename)
		if err == nil {
			_, err = io.Copy(part, src)
		}
		if closeErr := mw.Close(); err == nil {
			err = closeErr
		}
		_ = pw.CloseWithError(err)
	}()

	return pr, mw.FormDataContentType()
}

// GetStream returns a successful response whose body the caller must close.
// The stream client has no request timeout so callers can consume large
// downloads without the JSON client's short request deadline.
func (c *Client) GetStream(ctx context.Context, path string) (*http.Response, error) {
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.server+path, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "application/octet-stream")

		resp, err := c.streamHTTP.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 && c.retryOn401 != nil {
			newTok, retryErr := c.retryOn401(ctx)
			if retryErr != nil {
				return nil, retryErr
			}
			if newTok != "" {
				c.token = newTok
				continue
			}
		}
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(b)),
			RetryAfter: ParseRetryAfterSeconds(resp.Header.Get("Retry-After")),
		}
	}
	return nil, errors.New("stream request: exhausted 401 retries")
}

// PostMultipart streams one file field to path and decodes its JSON response.
// Multipart request bodies are single-pass and therefore are not retried.
func (c *Client) PostMultipart(ctx context.Context, path, field, filename string, src io.Reader, out any) error {
	body, contentType := newMultipartUploadBody(field, filename, src)
	defer func() { _ = body.Close() }()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.server+path, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", contentType)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(b)),
			RetryAfter: ParseRetryAfterSeconds(resp.Header.Get("Retry-After")),
		}
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
