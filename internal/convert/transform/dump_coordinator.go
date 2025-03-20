package transform

import (
	"context"
	"errors"
	"log/slog"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

type COption func(*Coordinator)

// Coordinator coordinates the conversion of chunk files to the desired format.
// It is used to convert files in parallel.
type Coordinator struct {
	idC  chan chunk.FileID
	errC chan error
	cvt  Converter
}

// WithIDChan allows to use an external ID channel.
func WithIDChan(idC chan chunk.FileID) COption {
	return func(c *Coordinator) {
		if idC != nil {
			c.idC = idC
		}
	}
}

// NewCoordinator creates a new Coordinator.
func NewCoordinator(ctx context.Context, cvt Converter, opts ...COption) *Coordinator {
	c := &Coordinator{
		cvt:  cvt,
		idC:  make(chan chunk.FileID),
		errC: make(chan error),
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

	for id := range c.idC {
		if err := c.cvt.Convert(ctx, id); err != nil {
			lg.Error("worker: conversion failed", "id", id, "error", err)
			c.errC <- err
		}
	}
}

// Wait closes the transformer.
func (s *Coordinator) Wait() (err error) {
	close(s.idC)
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

func (s *Coordinator) Transform(ctx context.Context, id chunk.FileID) error {
	select {
	case err := <-s.errC:
		return err
	default:
	}
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case s.idC <- id:
		// keep going
	}
	return nil
}
