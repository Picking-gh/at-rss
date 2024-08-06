/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"context"
	"time"

	"github.com/zyxar/argo/rpc"
)

// Aria2c handle the aria2c api request
type Aria2c struct {
	client rpc.Client
}

// NewAria2c return a new Aria2c object
func NewAria2c(url string, token string) (*Aria2c, error) {
	c, err := rpc.New(context.TODO(), url, token, 30*time.Second, nil)

	if err != nil {
		return nil, err
	}
	return &Aria2c{c}, nil
}

// Add add a new link to the aria2c server
func (a *Aria2c) Add(uri string) error {
	_, err := a.client.AddURI([]string{uri})

	if err != nil {
		return err
	}
	return nil
}

// CleanUp purges completed/error/removed downloads
func (a *Aria2c) CleanUp() {
	a.client.PurgeDownloadResult()
}

// Close closes the connection to the aria2 rpc interface
func (a *Aria2c) Close() {
	a.client.Close()
}
