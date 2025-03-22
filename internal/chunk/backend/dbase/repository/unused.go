//go:build ignore

package repository

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
)

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
