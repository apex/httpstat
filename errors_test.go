package httpstat_test

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/tj/assert"

	"github.com/apex/httpstat"
)

func TestResponse_errors(t *testing.T) {
	t.Run("max redirects exceeded", func(t *testing.T) {
		s := server(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "/")
			w.WriteHeader(302)
		})
		defer s.Close()

		_, err := httpstat.Request("GET", s.URL, nil, nil)
		assert.EqualError(t, err, "max redirects exceeded")
	})

	t.Run("tcp connection reset", func(t *testing.T) {
		sock, err := net.Listen("tcp", ":5555")
		assert.NoError(t, err, "listen")
		defer sock.Close()

		go func() {
			conn, err := sock.Accept()
			assert.NoError(t, err, "accept")

			conn.Close()
		}()

		_, err = httpstat.Request("GET", "http://localhost:5555", nil, nil)
		assert.EqualError(t, err, "connection reset by peer")
	})

	t.Run("malformed response", func(t *testing.T) {
		sock, err := net.Listen("tcp", ":5555")
		assert.NoError(t, err, "listen")
		defer sock.Close()

		go func() {
			conn, err := sock.Accept()
			assert.NoError(t, err, "accept")
			conn.Write([]byte("test"))
			conn.Close()
		}()

		_, err = httpstat.Request("GET", "http://localhost:5555", nil, nil)
		assert.EqualError(t, err, "malformed HTTP response")
	})

	// TODO: mock all this stuff

	t.Run("ssl", func(t *testing.T) {
		t.SkipNow()
		_, err := httpstat.Request("GET", "https://www.konbini.com", nil, nil)
		assert.EqualError(t, err, "SSL cert has expired")
	})

	t.Run("no host", func(t *testing.T) {
		_, err := httpstat.Request("GET", "http://whaefasdfadfasdfasdf.com", nil, nil)
		assert.EqualError(t, err, "no such host")
	})

	t.Run("invalid cert", func(t *testing.T) {
		_, err := httpstat.Request("GET", "https://radekstepan.com", nil, nil)
		assert.EqualError(t, err, "SSL cert is not valid for this domain")
	})

	t.Run("invalid cert 2", func(t *testing.T) {
		t.SkipNow()
		_, err := httpstat.Request("GET", "https://ec2-52-35-35-6.us-west-2.compute.amazonaws.com:8005/admin/", nil, nil)
		assert.EqualError(t, err, "SSL cert signed by unknown authority")
	})

	t.Run("invalid header name", func(t *testing.T) {
		h := make(http.Header)
		h.Set("foo/bar", "baz")
		_, err := httpstat.Request("GET", "https://apex.sh", h, nil)
		assert.EqualError(t, err, `invalid header field name "foo/bar"`)
	})

	t.Run("no https", func(t *testing.T) {
		_, err := httpstat.Request("GET", "https://tjholowaychuk.com", nil, nil)
		assert.EqualError(t, err, "connection refused")
	})

	t.Run("bad port", func(t *testing.T) {
		client := &http.Client{
			Timeout: 50 * time.Millisecond,
		}

		_, err := httpstat.RequestWithClient(client, "GET", "http://segment.com:9999", nil, nil)
		assert.EqualError(t, err, "timeout exceeded")
	})

	t.Run("invalid host", func(t *testing.T) {
		_, err := httpstat.Request("GET", "http://whatever", nil, nil)
		assert.EqualError(t, err, "no such host")
	})

	t.Run("timeout", func(t *testing.T) {
		client := &http.Client{
			Timeout: 50 * time.Millisecond,
		}

		_, err := httpstat.RequestWithClient(client, "GET", "http://apex.sh", nil, nil)
		assert.EqualError(t, err, "timeout exceeded")
	})
}
