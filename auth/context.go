package auth

import (
	"context"
	"errors"
)

type ctxKey int

const providerKey ctxKey = 0

func FromContext(ctx context.Context) (Provider, error) {
	prov, ok := ctx.Value(providerKey).(Provider)
	if !ok {
		return nil, errors.New("internal error:  no provider in context")
	}
	return prov, nil
}

func WithContext(pctx context.Context, p Provider) context.Context {
	return context.WithValue(pctx, providerKey, p)
}
