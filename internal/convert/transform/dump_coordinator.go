// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package transform

import (
	"context"
	"errors"
	"log/slog"

	"github.com/rusq/slack"
)

type COption func(*Coordinator)

// Coordinator coordinates the conversion of chunk files to the desired format.
// It is used to convert files in parallel.
type Coordinator struct {
	requests chan request
	errC     chan error
	cvt      Converter
}

// NewCoordinator creates a new Coordinator.
func NewCoordinator(ctx context.Context, cvt Converter, opts ...COption) *Coordinator {
	c := &Coordinator{
		cvt:      cvt,
		requests: make(chan request),
		errC:     make(chan error),
	}
	for _, opt := range opts {
		opt(c)
	}
	go c.worker(ctx)
	return c
}

func (c *Coordinator) worker(ctx context.Context) {
	defer close(c.errC)

	lg := slog.Default()

	for req := range c.requests {
		if err := c.cvt.Convert(ctx, req.channelID, req.threadTS); err != nil {
			lg.Error("worker: conversion failed", "id", req, "error", err)
			c.errC <- err
		}
	}
}

// Wait closes the transformer.
func (s *Coordinator) Wait() (err error) {
	close(s.requests)
	for e := range s.errC {
		if e != nil {
			err = errors.Join(err, e)
		}
	}
	return err
}

func (s *Coordinator) StartWithUsers(context.Context, []slack.User) error {
	// noop
	return nil
}

func (s *Coordinator) Transform(ctx context.Context, channelID, threadTS string) error {
	select {
	case err := <-s.errC:
		return err
	default:
	}
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case s.requests <- request{channelID: channelID, threadTS: threadTS}:
		// keep going
	}
	return nil
}
