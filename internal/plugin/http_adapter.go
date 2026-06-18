package plugin

import (
	"bytes"
	"io"
	"net/http"
)

// responseRecorder is a minimal http.ResponseWriter that captures the response
// produced by the embedded mux so the plugin can return it to the host as a
// ManagementResponse. It deliberately stays simpler than httptest's recorder:
// we never need flush, hijack, or trailers here.
type responseRecorder struct {
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{header: make(http.Header)}
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) Write(payload []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.body.Write(payload)
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.status = status
	r.wroteHeader = true
}

// bytesReader is split out so the cgo-importing file does not need bytes.
func bytesReader(payload []byte) io.Reader {
	return bytes.NewReader(payload)
}

// emptyReader returns a zero-length reader used when the management request
// has no body. Some net/http internals treat a typed-nil reader differently
// from an explicit empty reader, so this is safer than passing nil.
func emptyReader() io.Reader {
	return bytes.NewReader(nil)
}
