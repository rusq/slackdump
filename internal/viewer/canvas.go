package viewer

import (
	"io"

	"github.com/microcosm-cc/bluemonday"
)

var canvasPolicy = bluemonday.UGCPolicy()

// SanitizeCanvasDocument converts untrusted archived canvas markup into a
// complete, script-free document for both live and static renderers.
func SanitizeCanvasDocument(r io.Reader) string {
	b, err := io.ReadAll(r)
	if err != nil {
		return ""
	}
	return canvasDocument(string(canvasPolicy.SanitizeBytes(b)))
}

const canvasCSP = "default-src 'none'; img-src http: https: data:; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'self'"

func canvasDocument(content string) string {
	return "<!DOCTYPE html><html><head><meta charset=\"utf-8\"><meta http-equiv=\"Content-Security-Policy\" content=\"" + canvasCSP + "\"></head><body>" + content + "</body></html>"
}
