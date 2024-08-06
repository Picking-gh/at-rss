/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"log/slog"

	"github.com/siku2/arigo"
)

// Aria2c handle the aria2c api request
type Aria2c struct {
	client *arigo.Client
}

// NewAria2c return a new Aria2c object
func NewAria2c(url string, token string) (*Aria2c, error) {
	c, err := arigo.Dial(url, token)

	if err != nil {
		slog.Error("Failed to create aria2c rpc client.", "err", err)
		return nil, err
	}
	return &Aria2c{&c}, nil
}

// Add add a new link to the aria2c server
func (a *Aria2c) Add(uri string) error {
	_, err := a.client.AddURI(
		arigo.URIs(uri),
		nil)

	if err != nil {
		return err
	}
	return nil
}

// CleanUp purges completed/error/removed downloads
func (a *Aria2c) CleanUp() {
	a.client.PurgeDownloadResults()
}

// Close closes the connection to the aria2 rpc interface
func (a *Aria2c) Close() {
	a.client.Close()
}
