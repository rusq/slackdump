package repository

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"strings"

	"github.com/jmoiron/sqlx"
)

const (
	dbDriver = "sqlite"
	dbTag    = "db"
)

// PrepareExtContext is a combination of sqlx.PreparerContext and sqlx.ExtContext.
type PrepareExtContext interface {
	sqlx.PreparerContext
	sqlx.ExtContext
}

func newBindAddFn(buf *strings.Builder, binds *[]any) func(b bool, expr string, v any) {
	return func(b bool, expr string, v any) {
		if !b {
			return
		}
		buf.WriteString(expr)
		if v != nil {
			*binds = append(*binds, v)
		}
	}
}

func placeholders[T any](v []T) []string {
	s := make([]string, len(v))
	for i := range v {
		s[i] = "?"
	}
	return s
}

// orNull is a convenience function to set optional fields.
func orNull[T any](b bool, t T) *T {
	if b {
		return &t
	}
	return nil
}

var (
	marshal   = json.Marshal
	unmarshal = json.Unmarshal
)

// unmarshalt is a convenience function to unmarshal data into T.
func unmarshalt[T any](data []byte) (T, error) {
	var t T
	if err := unmarshal(data, &t); err != nil {
		return t, err
	}
	return t, nil
}

func marshalgz(a any) ([]byte, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := gz.Write(data); err != nil {
		gz.Close()
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const dictionary = "https://files.slack.com/files-tmb/" +
	`"blocks"` +
	`"client_msg_id"` +
	`"count"` +
	`"delete_original"` +
	`"elements"` +
	`"event_payload"` +
	`"event_type"` +
	`"fallback"` +
	`"false"` +
	`"latest_reply"` +
	`"message""` +
	`"metadata"` +
	`"parent_user_id"` +
	`"replace_original"` +
	`"reply_count"` +
	`"reply_users"` +
	`"rich_text""` +
	`"rich_text_section""` +
	`"text""` +
	`"thread_ts"` +
	`"thumb_` +
	`"true"` +
	`"type""` +
	`"url_private` +
	`"users"` +
	`:[{` +
	`null` +
	`}]}]}]},{`

func marshalflate(a any) ([]byte, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	// zw, err := flate.NewWriter(&buf, flate.BestCompression)
	zw, err := flate.NewWriterDict(&buf, flate.BestCompression, []byte(dictionary))
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(data); err != nil {
		zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func unmarshalflate(data []byte, v any) error {
	zr := flate.NewReaderDict(bytes.NewReader(data), []byte(dictionary))
	defer zr.Close()
	dec := json.NewDecoder(zr)
	return dec.Decode(v)
}

// slice is a convenience function to create a slice of T.
func slice[T any](s ...T) []T {
	return s
}
