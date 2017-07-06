package httpstat_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tj/assert"

	"github.com/apex/httpstat"
)

func server(h http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(h))
}

func assertDuration(t testing.TB, expected, actual time.Duration) {
	if actual < expected {
		t.Fatalf("%s is less than expected %s", actual, expected)
	}

	variance := 15 * time.Millisecond
	if actual > expected+variance {
		t.Fatalf("%s is more than expected %s + 15ms variance", actual, expected)
	}
}

func noRedirects(w http.ResponseWriter, r *http.Request) {
	time.Sleep(25 * time.Millisecond)
	w.Write([]byte("hello world"))
}

func redirects(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Location", "/bar")
		w.WriteHeader(302)
	case "/bar":
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Location", "/baz")
		w.WriteHeader(302)
	case "/baz":
		time.Sleep(25 * time.Millisecond)
		w.Write([]byte("hello world"))
	}
}

func TestResponse_ipAddress(t *testing.T) {
	res, err := httpstat.Request("GET", "http://52.34.209.136", nil, nil)
	assert.NoError(t, err, "request")
	assert.Equal(t, 200, res.Status(), "status")
}

func TestResponse_ipAddressWithPort(t *testing.T) {
	res, err := httpstat.Request("GET", "http://52.34.209.136:80", nil, nil)
	assert.NoError(t, err, "request")
	assert.Equal(t, 200, res.Status(), "status")
}

func TestResponse_hostWithPort(t *testing.T) {
	res, err := httpstat.Request("GET", "http://apex.sh:80", nil, nil)
	assert.NoError(t, err, "request")
	assert.Equal(t, 200, res.Status(), "status")
}

func TestResponse_hostWithPortArbitrary(t *testing.T) {
	res, err := httpstat.Request("GET", "http://tjholowaychuk.com:5000", nil, nil)
	assert.NoError(t, err, "request")
	assert.Equal(t, 200, res.Status(), "status")
}

func TestResponse(t *testing.T) {
	t.Run("with nil header", func(t *testing.T) {
		var val http.Header

		s := server(func(w http.ResponseWriter, r *http.Request) {
			val = r.Header
		})
		defer s.Close()

		res, err := httpstat.Request("GET", s.URL, nil, nil)
		assert.NoError(t, err, "request")

		assert.Equal(t, 200, res.Status(), "status")
		assert.Equal(t, val, http.Header{"User-Agent": []string{"Go-http-client/1.1"}, "Connection": []string{"close"}})
	})

	t.Run("with header", func(t *testing.T) {
		var val http.Header

		s := server(func(w http.ResponseWriter, r *http.Request) {
			val = r.Header
		})
		defer s.Close()

		header := http.Header{"User-Agent": []string{"ping/1.0"}}

		res, err := httpstat.Request("GET", s.URL, header, nil)
		assert.NoError(t, err, "request")

		assert.Equal(t, 200, res.Status(), "status")
		assert.Equal(t, val, http.Header{"User-Agent": []string{"ping/1.0"}, "Connection": []string{"close"}})
	})

	t.Run("with querystring", func(t *testing.T) {
		var val string

		s := server(func(w http.ResponseWriter, r *http.Request) {
			val = r.URL.Query().Get("foo")
		})
		defer s.Close()

		res, err := httpstat.Request("GET", s.URL+"?foo=bar", nil, nil)
		assert.NoError(t, err, "request")

		assert.Equal(t, 200, res.Status(), "status")
		assert.Equal(t, val, "bar")
	})

	t.Run("with basic auth", func(t *testing.T) {
		var val string

		s := server(func(w http.ResponseWriter, r *http.Request) {
			val = r.Header.Get("Authorization")
		})
		defer s.Close()

		url := strings.Replace(s.URL, "http://", "http://foo:bar@", 1)
		res, err := httpstat.Request("GET", url, nil, nil)
		assert.NoError(t, err, "request")

		assert.Equal(t, 200, res.Status(), "status")
		assert.Equal(t, val, "Basic Zm9vOmJhcg==")
	})
}

func TestResponse_TimeRedirects(t *testing.T) {
	t.Run("without redirects", func(t *testing.T) {
		s := server(noRedirects)
		defer s.Close()

		res, err := httpstat.Request("GET", s.URL, nil, nil)
		assert.NoError(t, err, "request")

		assert.Equal(t, 0, res.Redirects(), "redirects")
		assert.Equal(t, time.Duration(0), res.TimeRedirects(), "redirects")
	})

	t.Run("with redirects", func(t *testing.T) {
		s := server(redirects)
		defer s.Close()

		res, err := httpstat.Request("GET", s.URL, nil, nil)
		assert.NoError(t, err, "request")

		assert.Equal(t, 200, res.Status(), "status")
		assert.Equal(t, 2, res.Redirects(), "redirects")
		assertDuration(t, 100*time.Millisecond, res.TimeRedirects())
	})
}

func TestResponse_TimeTotal(t *testing.T) {
	t.Run("without redirects", func(t *testing.T) {
		s := server(noRedirects)
		defer s.Close()

		res, err := httpstat.Request("GET", s.URL, nil, nil)
		assert.NoError(t, err, "request")

		assertDuration(t, 25*time.Millisecond, res.TimeTotal(time.Now()))
	})

	t.Run("with redirects", func(t *testing.T) {
		s := server(redirects)
		defer s.Close()

		res, err := httpstat.Request("GET", s.URL, nil, nil)
		assert.NoError(t, err, "request")

		assert.Equal(t, 200, res.Status(), "status")
		assertDuration(t, 25*time.Millisecond, res.TimeTotal(time.Now()))
		assertDuration(t, 100*time.Millisecond, res.TimeRedirects())
	})
}

func TestResponse_TimeWait(t *testing.T) {
	t.Run("without redirects", func(t *testing.T) {
		s := server(noRedirects)
		defer s.Close()

		res, err := httpstat.Request("GET", s.URL, nil, nil)
		assert.NoError(t, err, "request")

		assertDuration(t, 25*time.Millisecond, res.TimeWait())
	})

	t.Run("with redirects", func(t *testing.T) {
		s := server(redirects)
		defer s.Close()

		res, err := httpstat.Request("GET", s.URL, nil, nil)
		assert.NoError(t, err, "request")

		assert.Equal(t, 200, res.Status(), "status")
		assertDuration(t, 25*time.Millisecond, res.TimeWait())
	})
}

func TestResponse_TimeTLS(t *testing.T) {
	t.Run("without redirects", func(t *testing.T) {
		res, err := httpstat.Request("GET", "https://apex.sh/ping/", nil, nil)
		assert.NoError(t, err, "request")

		assert.NotEmpty(t, res.TimeTLS())
	})

	t.Run("with redirects", func(t *testing.T) {
		res, err := httpstat.Request("GET", "http://apex.sh/", nil, nil)
		assert.NoError(t, err, "request")

		assert.NotEmpty(t, res.TimeTLS())
	})
}

func TestResponse_BodySize(t *testing.T) {
	t.Run("without redirects", func(t *testing.T) {
		s := server(redirects)
		defer s.Close()

		res, err := httpstat.Request("GET", s.URL, nil, nil)
		assert.NoError(t, err, "request")

		assert.Equal(t, 11, res.BodySize())
	})

	t.Run("with redirects", func(t *testing.T) {
		s := server(noRedirects)
		defer s.Close()

		res, err := httpstat.Request("GET", s.URL, nil, nil)
		assert.NoError(t, err, "request")

		assert.Equal(t, 11, res.BodySize())
	})
}

func TestResponse_Status(t *testing.T) {
	s := server(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})
	defer s.Close()

	res, err := httpstat.Request("GET", s.URL, nil, nil)
	assert.NoError(t, err, "request")

	assert.Equal(t, 500, res.Status(), "status")
}

func TestResponse_Header(t *testing.T) {
	s := server(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Foo", "foo")
		w.Header().Set("X-Bar", "bar")
		w.WriteHeader(500)
	})
	defer s.Close()

	res, err := httpstat.Request("GET", s.URL, nil, nil)
	assert.NoError(t, err, "request")

	assert.Equal(t, "foo", res.Header().Get("X-Foo"))
	assert.Equal(t, "bar", res.Header().Get("X-Bar"))
}

func TestResponse_TLS(t *testing.T) {
	t.Run("with HTTP", func(t *testing.T) {
		s := server(noRedirects)
		defer s.Close()

		res, err := httpstat.Request("GET", s.URL, nil, nil)
		assert.NoError(t, err, "request")

		assert.Equal(t, false, res.TLS())
	})

	t.Run("with HTTPS", func(t *testing.T) {
		res, err := httpstat.Request("GET", "https://apex.sh/ping/", nil, nil)
		assert.NoError(t, err, "request")

		assert.Equal(t, true, res.TLS())
	})

	t.Run("with HTTP to HTTPS redirects", func(t *testing.T) {
		res, err := httpstat.Request("GET", "http://apex.sh/", nil, nil)
		assert.NoError(t, err, "request")

		assert.Equal(t, true, res.TLS())
	})
}
