/*
 * Copyright (C) 2018 Aurélien Chabot <aurelien@chabot.fr>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"context"

	"github.com/hekmon/transmissionrpc/v2"
)

// Transmission handle the transmission api request
type Transmission struct {
	client *transmissionrpc.Client
}

// NewTransmission return a new Transmission object
func NewTransmission(host string, port uint16, user string, pswd string) (*Transmission, error) {

	t, err := transmissionrpc.New(host, user, pswd,
		&transmissionrpc.AdvancedConfig{
			Port: port,
		})
	if err != nil {
		return nil, err
	}
	return &Transmission{t}, nil
}

// Add add a new magnet link to the transmission server
func (t *Transmission) AddTorrent(magnet string) error {
	_, err := t.client.TorrentAdd(context.TODO(), transmissionrpc.TorrentAddPayload{
		Filename: &magnet,
	})
	if err != nil {
		return err
	}
	return nil
}

// Close do nothing but satisfy RpcClient interface
func (t *Transmission) Close() {}

// CleanUp do nothing but satisfy RpcClient interface
func (t *Transmission) CleanUp() {}
