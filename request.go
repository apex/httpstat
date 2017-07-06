package httpstat

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"time"
)

// TODO: distinct timeout errors for TLS etc

// DefaultMaxRedirects is the max number of redirects.
var DefaultMaxRedirects = 5

// DefaultClient used for requests.
var DefaultClient = &http.Client{
	CheckRedirect: checkRedirect,
	Timeout:       10 * time.Second,
	Transport: &http.Transport{
		DisableCompression: true,
		Proxy:              http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 0,
		}).DialContext,
		DisableKeepAlives:   true,
		MaxIdleConns:        10,
		TLSHandshakeTimeout: 5 * time.Second,
	},
}

// Check redirect.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > DefaultMaxRedirects {
		return ErrMaxRedirectsExceeded
	}

	return nil
}

// Size writer.
type sizeWriter int

// Write implementation.
func (w *sizeWriter) Write(b []byte) (int, error) {
	*w += sizeWriter(len(b))
	return len(b), nil
}

// Size of writes.
func (w sizeWriter) Size() int {
	return int(w)
}

// Response interface.
type Response interface {
	Status() int
	Redirects() int
	TLS() bool
	Header() http.Header
	HeaderSize() int
	BodySize() int
	TimeDNS() time.Duration
	TimeConnect() time.Duration
	TimeTLS() time.Duration
	TimeWait() time.Duration
	TimeResponse(time.Time) time.Duration
	TimeDownload(time.Time) time.Duration
	TimeTotal(time.Time) time.Duration
	TimeTotalWithRedirects(time.Time) time.Duration
	TimeRedirects() time.Duration
	Traces() []Trace
	Stats() *Stats
}

// Stats is an opaque struct which can be useful for JSON marshaling.
type Stats struct {
	Status                 int           `json:"status,omitempty"`
	Redirects              int           `json:"redirects,omitempty"`
	TLS                    bool          `json:"tls"`
	Header                 http.Header   `json:"header,omitempty"`
	HeaderSize             int           `json:"header_size,omitempty"`
	BodySize               int           `json:"body_size,omitempty"`
	TimeDNS                time.Duration `json:"time_dns"`
	TimeConnect            time.Duration `json:"time_connect"`
	TimeTLS                time.Duration `json:"time_tls"`
	TimeWait               time.Duration `json:"time_wait"`
	TimeResponse           time.Duration `json:"time_response"`
	TimeDownload           time.Duration `json:"time_download"`
	TimeTotal              time.Duration `json:"time_total"`
	TimeTotalWithRedirects time.Duration `json:"time_total_with_redirects,omitempty"`
	TimeRedirects          time.Duration `json:"time_redirects,omitempty"`
	Traces                 []*Stats      `json:"traces,omitempty"`
}

// Response struct.
type response struct {
	status     int
	traces     []Trace
	headerSize int
	header     http.Header
	bodySize   sizeWriter
}

// Stats returns a struct of stats.
func (r response) Stats() *Stats {
	now := time.Now()

	var traces []*Stats

	for _, t := range r.Traces() {
		traces = append(traces, t.Stats())
	}

	return &Stats{
		Status:                 r.Status(),
		Redirects:              r.Redirects(),
		TLS:                    r.TLS(),
		Header:                 r.Header(),
		HeaderSize:             r.HeaderSize(),
		BodySize:               r.BodySize(),
		TimeDNS:                r.TimeDNS(),
		TimeConnect:            r.TimeConnect(),
		TimeTLS:                r.TimeTLS(),
		TimeWait:               r.TimeWait(),
		TimeResponse:           r.TimeResponse(now),
		TimeDownload:           r.TimeDownload(now),
		TimeTotal:              r.TimeTotal(now),
		TimeTotalWithRedirects: r.TimeTotalWithRedirects(now),
		TimeRedirects:          r.TimeRedirects(),
		Traces:                 traces,
	}
}

// Status code.
func (r *response) Status() int {
	return r.status
}

// Last trace.
func (r *response) last() Trace {
	return r.traces[len(r.traces)-1]
}

// TLS implementation.
func (r *response) TLS() bool {
	return r.last().TLS()
}

// Redirects implementation.
func (r *response) Redirects() int {
	return len(r.traces) - 1
}

// BodySize implementation.
func (r *response) BodySize() int {
	return int(r.bodySize)
}

// HeaderSize implementation.
func (r *response) HeaderSize() int {
	return r.headerSize
}

// TimeDownload implementation.
func (r *response) TimeDownload(now time.Time) time.Duration {
	return r.last().TimeDownload(now)
}

// TimeResponse implementation.
func (r *response) TimeResponse(now time.Time) time.Duration {
	return r.last().TimeResponse(now)
}

// TimeDNS implementation.
func (r *response) TimeDNS() time.Duration {
	return r.last().TimeDNS()
}

// TimeConnect implementation.
func (r *response) TimeConnect() time.Duration {
	return r.last().TimeConnect()
}

// TimeTLS implementation.
func (r *response) TimeTLS() time.Duration {
	return r.last().TimeTLS()
}

// TimeWait implementation.
func (r *response) TimeWait() time.Duration {
	return r.last().TimeWait()
}

// TimeTotal implementation.
func (r *response) TimeTotal(now time.Time) time.Duration {
	return r.last().TimeTotal(now)
}

// TimeTotalWithRedirects implementation.
func (r *response) TimeTotalWithRedirects(now time.Time) time.Duration {
	return r.traces[0].TimeTotal(now)
}

// TimeRedirects implementation.
func (r *response) TimeRedirects() time.Duration {
	if len(r.traces) == 1 {
		return 0
	}

	first := r.traces[0]
	last := r.traces[len(r.traces)-1]

	return last.Start().Sub(first.Start())
}

// Header implementation.
func (r *response) Header() http.Header {
	return r.header
}

// Traces implementation.
func (r *response) Traces() []Trace {
	return r.traces
}

// RequestWithClient performs a traced request.
func RequestWithClient(client *http.Client, method, uri string, header http.Header, body io.Reader) (Response, error) {
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}

	for name, field := range header {
		for _, v := range field {
			req.Header.Set(name, v)
		}
	}

	var out response
	req = req.WithContext(WithTraces(req.Context(), &out.traces))

	res, err := client.Do(req)
	if err != nil {
		return nil, normalizeError(err)
	}
	defer res.Body.Close()

	out.status = res.StatusCode

	if _, err := io.Copy(&out.bodySize, res.Body); err != nil {
		return nil, err
	}

	var resHeader bytes.Buffer
	res.Header.Write(&resHeader)
	out.header = res.Header
	out.headerSize = resHeader.Len()

	return &out, nil
}

// Request performs a traced request.
func Request(method, uri string, header http.Header, body io.Reader) (Response, error) {
	return RequestWithClient(DefaultClient, method, uri, header, body)
}
