package transform

import (
	"context"
	"errors"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2/internal/chunk"
)

type COption func(*Coordinator)

// Coordinator is the transform coordinator.
type Coordinator struct {
	idC  chan chunk.FileID
	errC chan error
	cvt  Converter
}

// WithIDC allows to use an external ID channel.
func WithIDChan(idC chan chunk.FileID) COption {
	return func(c *Coordinator) {
		if idC != nil {
			c.idC = idC
		}
	}
}

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

	for id := range c.idC {
		if err := c.cvt.Convert(ctx, id); err != nil {
			dlog.Printf("error converting %q: %v", id, err)
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

func (s *Coordinator) Transform(ctx context.Context, id chunk.FileID) error {
	select {
	case err := <-s.errC:
		return err
	default:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.idC <- id:
		// keep going
	}
	return nil
}
