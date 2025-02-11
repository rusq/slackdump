package repository

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
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

var marshal = marshalflate

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

func marshalzlib(a any) ([]byte, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
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

func marshalflate(a any) ([]byte, error) {
	const dictionary = "https://files.slack.com/files-tmb/" +
		`:[{` +
		`:null},` +
		`"blocks"` +
		`"client_msg_id"` +
		`"delete_original"` +
		`elements` +
		`event_payload` +
		`event_type` +
		`fallback` +
		`"message""` +
		`"metadata"` +
		`"parent_user_id"` +
		`"replace_original"` +
		`"rich_text_section""` +
		`"rich_text""` +
		`"text""` +
		`thread_ts` +
		`"thumb_` +
		`"type""` +
		`"url_private` +
		`}]}]}]},{`

	data, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	//zw, err := flate.NewWriter(&buf, flate.BestCompression)
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

func marshallzw(a any) ([]byte, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw := lzw.NewWriter(&buf, lzw.LSB, 8)
	if _, err := zw.Write(data); err != nil {
		zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
