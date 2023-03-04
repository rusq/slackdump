package auth

import (
	"context"
	"errors"
)

type ctxKey int

const providerKey ctxKey = 0

var ErrNoProvider = errors.New("internal error:  no provider in context")

// FromContext returns the auth provider from the context.
func FromContext(ctx context.Context) (Provider, error) {
	prov, ok := ctx.Value(providerKey).(Provider)
	if !ok {
		return nil, ErrNoProvider
	}
	return prov, nil
}

// WithContext returns context with auth provider.
func WithContext(pctx context.Context, p Provider) context.Context {
	return context.WithValue(pctx, providerKey, p)
}
