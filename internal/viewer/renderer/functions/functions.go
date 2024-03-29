// Package functions provides shared template functions.
package functions

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"mime"
	"strings"
	"time"
)

var FuncMap = template.FuncMap{
	"epoch":    Epoch,
	"mimetype": Mimetype,
}

func Epoch(ts json.Number) string {
	if ts == "" {
		return ""
	}
	t, err := ts.Int64()
	if err != nil {
		tf, err := ts.Float64()
		if err != nil {
			slog.Debug("epoch Float64 error, out of conversion options", "err", err, "ts", ts)
			return ts.String()
		}
		t = int64(tf)
	}
	return time.Unix(t, 0).Local().Format(time.DateTime)
}

func Mimetype(mt string) string {
	mm, _, err := mime.ParseMediaType(mt)
	if err != nil || mt == "" {
		slog.Debug("mimetype", "err", err, "mimetype", mt)
		return "application"
	}
	slog.Debug("mimetype", "t", mm, "mimetype", mt)
	t, _, found := strings.Cut(mm, "/")
	if !found {
		return "application"
	}
	return t
}
