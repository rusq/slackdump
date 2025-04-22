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

type request struct {
	channelID, threadTS string
	threadOnly          bool
}

func (s *Coordinator) Transform(ctx context.Context, channelID, threadTS string, threadOnly bool) error {
	select {
	case err := <-s.errC:
		return err
	default:
	}
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case s.requests <- request{channelID: channelID, threadTS: threadTS, threadOnly: threadOnly}:
		// keep going
	}
	return nil
}
