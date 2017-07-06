package httpstat

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http/httptrace"
	"time"
)

// TODO: TimeResponse etc should have the end time stored,
// currently only relevant to the final request

// TODO: store more request information for redirects,
// such as the url :)

// Trace results.
type Trace interface {
	Address() string
	TLS() bool
	Start() time.Time
	TimeDNS() time.Duration
	TimeConnect() time.Duration
	TimeTLS() time.Duration
	TimeWait() time.Duration
	TimeResponse(time.Time) time.Duration
	TimeDownload(time.Time) time.Duration
	TimeTotal(time.Time) time.Duration
	Stats() *Stats
}

type trace struct {
	addr string
	tls  bool

	start     time.Time
	dnsStart  time.Time
	dnsEnd    time.Time
	tcpStart  time.Time
	tcpEnd    time.Time
	tlsStart  time.Time
	tlsEnd    time.Time
	waitStart time.Time
	waitEnd   time.Time
}

// TLS implementation.
func (t *trace) TLS() bool {
	return t.tls
}

// Address implementation.
func (t *trace) Address() string {
	return t.addr
}

// Start implementation.
func (t *trace) Start() time.Time {
	return t.start
}

// TimeDNS implementation.
func (t *trace) TimeDNS() time.Duration {
	return t.dnsEnd.Sub(t.dnsStart)
}

// TimeConnect implementation.
func (t *trace) TimeConnect() time.Duration {
	return t.tcpEnd.Sub(t.tcpStart)
}

// TimeTLS implementation.
func (t *trace) TimeTLS() time.Duration {
	return t.tlsEnd.Sub(t.tlsStart)
}

// TimeWait implementation.
func (t *trace) TimeWait() time.Duration {
	return t.waitEnd.Sub(t.waitStart)
}

// TimeDownload implementation.
func (t *trace) TimeDownload(now time.Time) time.Duration {
	return now.Sub(t.waitEnd)
}

// TimeResponse implementation.
func (t *trace) TimeResponse(now time.Time) time.Duration {
	return now.Sub(t.waitStart)
}

// TimeTotal implementation.
func (t *trace) TimeTotal(now time.Time) time.Duration {
	return now.Sub(t.start)
}

// WithTraces traces request timings.
func WithTraces(ctx context.Context, traces *[]Trace) context.Context {
	var t *trace

	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GetConn: func(addr string) {
			t = &trace{}
			t.start = time.Now()
			t.addr = addr
		},

		GotConn: func(info httptrace.GotConnInfo) {
			if info.Reused {
				t = &trace{}
				t.start = time.Now()
			}
			*traces = append(*traces, t)
		},

		DNSStart: func(info httptrace.DNSStartInfo) {
			t.dnsStart = time.Now()
		},

		DNSDone: func(info httptrace.DNSDoneInfo) {
			t.dnsEnd = time.Now()
		},

		ConnectStart: func(network, addr string) {
			t.tcpStart = time.Now()
		},

		ConnectDone: func(network, addr string, err error) {
			t.tcpEnd = time.Now()
		},

		TLSHandshakeStart: func() {
			t.tls = true
			t.tlsStart = time.Now()
		},

		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			t.tlsEnd = time.Now()
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			t.waitStart = time.Now()
		},

		GotFirstResponseByte: func() {
			t.waitEnd = time.Now()
		},
	})
}

// Stats returns a struct of stats.
func (t trace) Stats() *Stats {
	now := time.Now()

	return &Stats{
		TLS:          t.TLS(),
		TimeDNS:      t.TimeDNS(),
		TimeConnect:  t.TimeConnect(),
		TimeTLS:      t.TimeTLS(),
		TimeWait:     t.TimeWait(),
		TimeResponse: t.TimeResponse(now),
		TimeDownload: t.TimeDownload(now),
		TimeTotal:    t.TimeTotal(now),
	}
}

// Millisecond formatter.
func ms(d time.Duration) string {
	return fmt.Sprintf("%.0fms", float64(d)/float64(time.Millisecond))
}
