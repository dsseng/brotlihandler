package brotlihandler

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
)

// BrotliHandler creates a new handler which compresses
// the data using Brotli when client supports it
func BrotliHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if acceptsBr(r) {
			w.Header().Set("Vary", "Accept-Encoding")
			bw := brotli.NewWriter(w)
			brResponseWriter := &BrotliResponseWriter{
				responseWriter: w,
				brotliWriter:   *bw,
			}
			defer brResponseWriter.Close()

			h.ServeHTTP(brResponseWriter, r)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

// acceptsBr returns true if the given HTTP request indicates that it will
// accept a Brotli response.
func acceptsBr(r *http.Request) bool {
	acceptedEncodings := parseEncodings(r.Header.Get("Accept-Encoding"))
	return acceptedEncodings["br"] > 0.0
}

type codings map[string]float64

// parseEncodings attempts to parse a list of codings, per RFC 2616, as might
// appear in an Accept-Encoding header. It returns a map of content-codings to
// quality values.
//
// See: http://tools.ietf.org/html/rfc2616#section-14.3.
func parseEncodings(s string) codings {
	c := make(codings)

	for _, ss := range strings.Split(s, ",") {
		coding, qvalue, err := parseCoding(ss)

		if err == nil {
			c[coding] = qvalue
		}
	}

	return c
}

// parseCoding parses a single conding (content-coding with an optional qvalue),
// as might appear in an Accept-Encoding header. It attempts to forgive minor
// formatting errors.
func parseCoding(s string) (coding string, qvalue float64, err error) {
	for n, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		qvalue = 1.0

		if n == 0 {
			coding = strings.ToLower(part)
		} else if strings.HasPrefix(part, "q=") {
			qvalue, err = strconv.ParseFloat(strings.TrimPrefix(part, "q="), 64)

			if qvalue < 0.0 {
				qvalue = 0.0
			} else if qvalue > 1.0 {
				qvalue = 1.0
			}
		}
	}

	if coding == "" {
		err = fmt.Errorf("empty content-coding")
	}

	return
}

// BrotliResponseWriter provides an http.ResponseWriter
// interface, which compresses bytes with Brotli before
// writing them to the underlying response. This doesn't
// close the writers, so don't forget to do that.
type BrotliResponseWriter struct {
	responseWriter http.ResponseWriter
	brotliWriter   brotli.Writer
}

// Header returns the header map that will be sent by
// WriteHeader. The Header map also is the mechanism with which
// Handlers can set HTTP trailers.
//
// Changing the header map after a call to WriteHeader (or
// Write) has no effect unless the modified headers are
// trailers.
//
// There are two ways to set Trailers. The preferred way is to
// predeclare in the headers which trailers you will later
// send by setting the "Trailer" header to the names of the
// trailer keys which will come later. In this case, those
// keys of the Header map are treated as if they were
// trailers. See the example. The second way, for trailer
// keys not known to the Handler until after the first Write,
// is to prefix the Header map keys with the TrailerPrefix
// constant value. See TrailerPrefix.
//
// To suppress automatic response headers (such as "Date"), set
// their value to nil.
func (w *BrotliResponseWriter) Header() http.Header {
	return w.responseWriter.Header()
}

// WriteHeader sends an HTTP response header with the provided
// status code.
//
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to
// send error codes.
//
// The provided code must be a valid HTTP 1xx-5xx status code.
// Only one header may be written. Go does not currently
// support sending user-defined 1xx informational headers,
// with the exception of 100-continue response header that the
// Server sends automatically when the Request.Body is read.
func (w *BrotliResponseWriter) WriteHeader(c int) {
	w.responseWriter.WriteHeader(c)
}

// Write writes the data to the connection as part of an HTTP reply.
//
// If WriteHeader has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data. If the Header
// does not contain a Content-Type line, Write adds a Content-Type set
// to the result of passing the initial 512 bytes of written data to
// DetectContentType. Additionally, if the total size of all written
// data is under a few KB and there are no Flush calls, the
// Content-Length header is added automatically.
//
// Depending on the HTTP protocol version and the client, calling
// Write or WriteHeader may prevent future reads on the
// Request.Body. For HTTP/1.x requests, handlers should read any
// needed request body data before writing the response. Once the
// headers have been flushed (due to either an explicit Flusher.Flush
// call or writing enough data to trigger a flush), the request body
// may be unavailable. For HTTP/2 requests, the Go HTTP server permits
// handlers to continue to read the request body while concurrently
// writing the response. However, such behavior may not be supported
// by all HTTP/2 clients. Handlers should read before writing if
// possible to maximize compatibility.
func (w *BrotliResponseWriter) Write(b []byte) (int, error) {
	h := w.responseWriter.Header()
	h.Add("Content-Encoding", "br")
	if h.Get("Content-Type") == "" {
		h.Set("Content-Type", http.DetectContentType(b))
	}
	h.Del("Content-Length")

	return w.brotliWriter.Write(b)
}

// Flush sends any buffered data to the client.
func (w *BrotliResponseWriter) Flush() {
	w.brotliWriter.Flush()

	if fw, ok := w.responseWriter.(http.Flusher); ok {
		fw.Flush()
	}
}

// Hijack lets the caller take over the connection.
// After a call to Hijack the HTTP server library
// will not do anything else with the connection.
//
// It becomes the caller's responsibility to manage
// and close the connection.
//
// The returned net.Conn may have read or write deadlines
// already set, depending on the configuration of the
// Server. It is the caller's responsibility to set
// or clear those deadlines as needed.
//
// The returned bufio.Reader may contain unprocessed buffered
// data from the client.
//
// After a call to Hijack, the original Request.Body must not
// be used. The original Request's Context remains valid and
// is not canceled until the Request's ServeHTTP method
// returns.
func (w *BrotliResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.responseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("http.Hijacker interface is not supported")
}

// Close immediately closes all active net.Listeners and any connections in state StateNew, StateActive, or StateIdle. For a graceful shutdown, use Shutdown.
// Close does not attempt to close (and does not even know about) any hijacked connections, such as WebSockets.
// Close returns any error returned from closing the Server's underlying Listener(s).
func (w *BrotliResponseWriter) Close() error {
	return w.brotliWriter.Close()
}

// CloseNotify returns a channel that receives at most a
// single value (true) when the client connection has gone
// away.
//
// CloseNotify may wait to notify until Request.Body has been
// fully read.
//
// After the Handler has returned, there is no guarantee
// that the channel receives a value.
//
// If the protocol is HTTP/1.1 and CloseNotify is called while
// processing an idempotent request (such a GET) while
// HTTP/1.1 pipelining is in use, the arrival of a subsequent
// pipelined request may cause a value to be sent on the
// returned channel. In practice HTTP/1.1 pipelining is not
// enabled in browsers and not seen often in the wild. If this
// is a problem, use HTTP/2 or only use CloseNotify on methods
// such as POST.
func (w *BrotliResponseWriter) CloseNotify() <-chan bool {
	return w.responseWriter.(http.CloseNotifier).CloseNotify()
}
