/*
 * Copyright (C) 2018 Aurélien Chabot <aurelien@chabot.fr>
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
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

const transmissionSessionIDHeader = "X-Transmission-Session-Id"

// Transmission handle the transmission api request
type Transmission struct {
	rpcURL     string
	httpClient *http.Client
	ctx        context.Context
	sessionID  string
	user       string
	password   string
	mu         sync.Mutex // Protects sessionID
}

type transmissionRequest struct {
	Method    string `json:"method"`
	Arguments any    `json:"arguments"`
	Tag       int    `json:"tag,omitempty"`
}

type transmissionResponse struct {
	Result    string `json:"result"` // "success" or error string
	Arguments any    `json:"arguments,omitempty"`
	Tag       int    `json:"tag,omitempty"`
}

type torrentAddPayload struct {
	Filename string `json:"filename"` // Can be magnet link or torrent file path/URL
}

// Define structures for torrent-get response
type torrentGetResponse struct {
	Torrents []torrentDetails `json:"torrents"`
}

type torrentDetails struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Status      int     `json:"status"` // 0: stopped, 1: check pending, 2: checking, 3: download pending, 4: downloading, 5: seed pending, 6: seeding
	IsFinished  bool    `json:"isFinished"`
	PercentDone float64 `json:"percentDone"` // Use this as a fallback if isFinished is not reliable or status is ambiguous
}

// Define structure for torrent-remove arguments
type torrentRemovePayload struct {
	IDs             []int `json:"ids"`
	DeleteLocalData bool  `json:"delete-local-data"`
}

// NewTransmission returns a new Transmission object.
// It expects rpcUrl to be a valid http or https URL.
func NewTransmission(ctx context.Context, rpcUrl string, user string, pswd string) (*Transmission, error) {
	if rpcUrl == "" {
		return nil, fmt.Errorf("transmission RPC URL cannot be empty")
	}
	// Basic validation
	if !strings.HasPrefix(rpcUrl, "http://") && !strings.HasPrefix(rpcUrl, "https://") {
		return nil, fmt.Errorf("invalid transmission RPC URL scheme in %q: must be http or https", rpcUrl)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	t := &Transmission{
		rpcURL:     rpcUrl, // Use the validated input URL directly
		httpClient: client,
		ctx:        ctx,
		user:       user,
		password:   pswd,
	}

	// Initial request to get the session ID (will likely fail with 409)
	_, err := t.call("session-get", nil) // Use a simple method like session-get
	if err != nil && t.sessionID == "" { // Expecting a 409 error which sets the session ID
		// Check if it was a session ID error specifically, otherwise return the error
		// This part is simplified; a real implementation might parse the 409 response better.
		// If sessionID is still empty after the call attempt, something else went wrong.
		return nil, fmt.Errorf("failed to get initial transmission session ID: %w", err)
	}
	if t.sessionID == "" {
		return nil, fmt.Errorf("could not obtain transmission session ID from %s", rpcUrl)
	}

	return t, nil
}

func (t *Transmission) call(method string, args any) (*transmissionResponse, error) {
	t.mu.Lock()
	currentSessionID := t.sessionID
	t.mu.Unlock()

	tag := rand.Int()
	reqPayload := transmissionRequest{
		Method:    method,
		Arguments: args,
		Tag:       tag,
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transmission request: %w", err)
	}

	req, err := http.NewRequestWithContext(t.ctx, "POST", t.rpcURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create transmission request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if currentSessionID != "" {
		req.Header.Set(transmissionSessionIDHeader, currentSessionID)
	}
	if t.user != "" || t.password != "" {
		req.SetBasicAuth(t.user, t.password)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute transmission request (%s): %w", method, err)
	}
	defer resp.Body.Close()

	// Handle 409 Conflict for session ID
	if resp.StatusCode == http.StatusConflict {
		newSessionID := resp.Header.Get(transmissionSessionIDHeader)
		if newSessionID == "" {
			return nil, fmt.Errorf("transmission request (%s) failed with 409 Conflict but no new session ID provided", method)
		}
		t.mu.Lock()
		t.sessionID = newSessionID
		t.mu.Unlock()
		return t.call(method, args)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("transmission request (%s) failed with status: %s", method, resp.Status)
	}

	var respPayload transmissionResponse
	if err := json.NewDecoder(resp.Body).Decode(&respPayload); err != nil {
		return nil, fmt.Errorf("failed to decode transmission response (%s): %w", method, err)
	}

	if respPayload.Tag != tag {
		return nil, fmt.Errorf("transmission response tag mismatch (expected %d, got %d)", tag, respPayload.Tag)
	}

	if respPayload.Result != "success" {
		return nil, fmt.Errorf("transmission rpc error (%s): %s", method, respPayload.Result)
	}

	return &respPayload, nil
}

// AddTorrent adds a new magnet link to the transmission server
func (t *Transmission) AddTorrent(magnet string) error {
	payload := torrentAddPayload{
		Filename: magnet,
	}
	_, err := t.call("torrent-add", payload)
	return err
}

// CloseRpc closes idle connections.
func (t *Transmission) CloseRpc() {
	t.httpClient.CloseIdleConnections()
}

// CleanUp removes completed and stopped torrents from Transmission (without deleting data).
func (t *Transmission) CleanUp() {
	getArgs := struct {
		Fields []string `json:"fields"`
	}{
		Fields: []string{"id", "name", "status", "isFinished", "percentDone"},
	}

	resp, err := t.call("torrent-get", getArgs)
	if err != nil {
		slog.Warn("Transmission CleanUp: Failed to get torrent list", "error", err)
		return
	}

	var torrentList torrentGetResponse
	// Arguments in the response are often returned as a map[string]any
	// We need to marshal the relevant part back to JSON and then unmarshal into our struct
	argsMap, ok := resp.Arguments.(map[string]any)
	if !ok {
		slog.Warn("Transmission CleanUp: Unexpected format for torrent-get arguments")
		return
	}
	argsJSON, err := json.Marshal(argsMap)
	if err != nil {
		slog.Warn("Transmission CleanUp: Failed to marshal torrent-get arguments", "error", err)
		return
	}
	if err := json.Unmarshal(argsJSON, &torrentList); err != nil {
		slog.Warn("Transmission CleanUp: Failed to unmarshal torrent list", "error", err)
		return
	}

	var idsToRemove []int
	for _, torrent := range torrentList.Torrents {
		isDone := torrent.IsFinished || torrent.PercentDone >= 1.0
		isStopped := torrent.Status == 0

		if isDone && isStopped {
			idsToRemove = append(idsToRemove, torrent.ID)
			slog.Debug("Transmission CleanUp: Marking torrent for removal", "id", torrent.ID, "name", torrent.Name)
		}
	}

	if len(idsToRemove) > 0 {
		removeArgs := torrentRemovePayload{
			IDs:             idsToRemove,
			DeleteLocalData: false,
		}
		_, err := t.call("torrent-remove", removeArgs)
		if err != nil {
			slog.Warn("Transmission CleanUp: Failed to remove torrents", "ids", idsToRemove, "error", err)
		} else {
			slog.Info("Transmission CleanUp: Successfully removed completed and stopped torrents", "count", len(idsToRemove))
		}
	} else {
		slog.Debug("Transmission CleanUp: No completed and stopped torrents found to remove.")
	}
}
