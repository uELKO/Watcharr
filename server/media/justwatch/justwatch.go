// Package justwatch is a client for JustWatch's public GraphQL API
// (the same one justwatch.com's own frontend uses). It is NOT an official
// or documented API - JustWatch's real, contract-based partner API
// (apis.justwatch.com's "Content Partner API") requires a paid business
// agreement and isn't practical for a personal/self-hosted use case. This
// client only asks for the minimum fields we actually use (title, TMDB id,
// chart rank/trend), reducing the surface area exposed to schema changes,
// and can fail entirely if JustWatch changes their (undocumented) schema -
// callers should treat errors from this package as "feature unavailable
// right now", not fatal.
package justwatch

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	gocache "github.com/robfig/go-cache"
)

const apiURL = "https://apis.justwatch.com/graphql"

// ContentStore caches JustWatch responses, same pattern as tmdb.ContentStore -
// this is an unofficial/undocumented API, so caching also means we're not
// hammering it more than necessary.
var ContentStore = gocache.New(time.Hour*24, time.Minute)

type JustWatch struct {
	client *http.Client
}

func NewJustWatch() *JustWatch {
	return &JustWatch{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// req sends one GraphQL query and unmarshals the "data" field of the
// response into resp.
func (j *JustWatch) req(operationName string, variables map[string]any, query string, resp any) error {
	body, err := json.Marshal(map[string]any{
		"operationName": operationName,
		"variables":     variables,
		"query":         query,
	})
	if err != nil {
		return err
	}
	httpResp, err := j.client.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Error("justwatch req: request failed", "operation", operationName, "error", err)
		return errors.New("request failed")
	}
	defer httpResp.Body.Close()
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}
	if httpResp.StatusCode != 200 {
		slog.Error("justwatch req: non-200 status", "operation", operationName, "status", httpResp.StatusCode, "body", string(respBody))
		return errors.New("request failed")
	}
	var wrapper struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		slog.Error("justwatch req: failed to unmarshal response", "operation", operationName, "error", err)
		return errors.New("invalid response")
	}
	if len(wrapper.Errors) > 0 {
		slog.Error("justwatch req: GraphQL errors", "operation", operationName, "errors", wrapper.Errors)
		return errors.New("graphql error")
	}
	if err := json.Unmarshal(wrapper.Data, resp); err != nil {
		slog.Error("justwatch req: failed to unmarshal data", "operation", operationName, "error", err)
		return errors.New("invalid response")
	}
	return nil
}
