/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// Aria2c handle the aria2c api request
type Aria2c struct {
	rpcURL     string
	httpClient *http.Client
	ctx        context.Context
	rpcToken   string
}

type aria2Request struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type aria2Response struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Result  any         `json:"result,omitempty"`
	Error   *aria2Error `json:"error,omitempty"`
}

type aria2Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *aria2Error) Error() string {
	return fmt.Sprintf("aria2 rpc error (%d): %s", e.Code, e.Message)
}

// NewAria2c returns a new Aria2c object.
// It expects rpcUrl to be a valid http or https URL.
func NewAria2c(ctx context.Context, rpcUrl string, token string) (*Aria2c, error) {
	if rpcUrl == "" {
		return nil, fmt.Errorf("aria2c RPC URL cannot be empty")
	}

	// Basic validation if it looks like a URL (doesn't need full parsing now)
	if !strings.HasPrefix(rpcUrl, "http://") && !strings.HasPrefix(rpcUrl, "https://") {
		return nil, fmt.Errorf("invalid aria2c RPC URL scheme in %q: must be http or https", rpcUrl)
	}

	client := &http.Client{
		Timeout: 30 * time.Second, // Keep the original timeout
	}
	a := &Aria2c{
		rpcURL:     rpcUrl,
		rpcToken:   "token:" + token, // Aria2 expects "token:" prefix
		httpClient: client,
		ctx:        ctx,
	}

	return a, nil
}

func (a *Aria2c) call(method string, params []any) (*aria2Response, error) {
	actualParams := append([]any{a.rpcToken}, params...)

	reqPayload := aria2Request{
		Jsonrpc: "2.0",
		ID:      fmt.Sprintf("at-rss-%d", rand.Int()), // Simple unique ID
		Method:  method,
		Params:  actualParams,
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal aria2c request: %w", err)
	}

	req, err := http.NewRequestWithContext(a.ctx, "POST", a.rpcURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create aria2c request to %s: %w", a.rpcURL, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		// Include the target URL in the error message for clarity
		return nil, fmt.Errorf("failed to execute aria2c request (%s) to %s: %w", method, a.rpcURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aria2c request (%s) to %s failed with status: %s", method, a.rpcURL, resp.Status)
	}

	var respPayload aria2Response
	if err := json.NewDecoder(resp.Body).Decode(&respPayload); err != nil {
		return nil, fmt.Errorf("failed to decode aria2c response (%s) from %s: %w", method, a.rpcURL, err)
	}

	if respPayload.Error != nil {
		return nil, respPayload.Error
	}

	return &respPayload, nil
}

// AddTorrent adds a new torrent URI to the aria2c server
func (a *Aria2c) AddTorrent(uri string) error {
	// AddURI expects a slice of URIs and options map
	// We pass an empty options map {}
	_, err := a.call("aria2.addUri", []any{[]string{uri}, map[string]string{}})
	return err
}

// CleanUp purges completed/error/removed downloads
func (a *Aria2c) CleanUp() {
	// PurgeDownloadResult takes no extra parameters
	_, _ = a.call("aria2.purgeDownloadResult", []any{})
	// Ignore error for cleanup, best effort
}

// CloseRpc closes the underlying http client idle connections
func (a *Aria2c) CloseRpc() {
	a.httpClient.CloseIdleConnections()
}

// GetActiveDownloads returns the current download status from aria2c
func (a *Aria2c) GetActiveDownloads() ([]DownloadStatus, error) {
	// Get active downloads
	activeResp, err := a.call("aria2.tellActive", []any{})
	if err != nil {
		return nil, fmt.Errorf("failed to get active downloads: %w", err)
	}

	// Get waiting downloads (including paused)
	waitingResp, err := a.call("aria2.tellWaiting", []any{0, 1000}) // Get first 1000 waiting items
	if err != nil {
		return nil, fmt.Errorf("failed to get waiting downloads: %w", err)
	}

	var statuses []DownloadStatus

	// Process active downloads
	if activeList, ok := activeResp.Result.([]any); ok {
		for _, item := range activeList {
			if download, ok := item.(map[string]any); ok {
				status := a.parseDownloadStatus(download)
				statuses = append(statuses, status)
			}
		}
	}

	// Process waiting downloads
	if waitingList, ok := waitingResp.Result.([]any); ok {
		for _, item := range waitingList {
			if download, ok := item.(map[string]any); ok {
				status := a.parseDownloadStatus(download)
				statuses = append(statuses, status)
			}
		}
	}

	return statuses, nil
}

func (a *Aria2c) parseDownloadStatus(download map[string]any) DownloadStatus {
	status := DownloadStatus{
		ID:          fmt.Sprintf("%v", download["gid"]),
		Name:        fmt.Sprintf("%v", download["bittorrent"]), // TODO: parse name from bittorrent info
		Downloader:  "aria2c",
		PercentDone: 0,
	}

	// Parse status
	switch fmt.Sprintf("%v", download["status"]) {
	case "active":
		status.Status = "downloading"
	case "waiting":
		status.Status = "stopped"
	case "paused":
		status.Status = "stopped"
	default:
		status.Status = "error"
	}

	// Parse progress
	if total, ok := download["totalLength"].(float64); ok {
		if completed, ok := download["completedLength"].(float64); ok && total > 0 {
			status.PercentDone = completed / total
			if status.PercentDone >= 1.0 {
				status.IsFinished = true
				status.Status = "seeding"
			}
		}
	}

	return status
}
