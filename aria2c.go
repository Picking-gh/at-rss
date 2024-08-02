/*
 * Copyright (C) 2024 Picking-gh <picking@woft.name>
 *
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"log"

	"github.com/siku2/arigo"
)

// Aria2c handle the aria2c api request
type Aria2c struct {
	client *arigo.Client
}

// NewAria2c return a new Aria2c object
func NewAria2c(url string, token string) *Aria2c {
	c, err := arigo.Dial(url, token)

	if err != nil {
		log.Fatal(err)
	}
	return &Aria2c{&c}
}

// Add add a new magnet link to the aria2c server
func (c *Aria2c) Add(magnet string) error {
	_, err := c.client.Download(arigo.URIs(magnet), nil)

	if err != nil {
		return err
	}
	return nil
}
