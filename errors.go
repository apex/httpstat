package httpstat

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
)

// TODO: moar user-friendly mappings...

// Errors.
var (
	ErrMaxRedirectsExceeded = errors.New("max redirects exceeded")
	ErrTimeoutExceeded      = errors.New("timeout exceeded")
)

// Normalize the given error.
func normalizeError(err error) error {
	if err, ok := err.(*url.Error); ok {
		return urlError(err)
	}

	return err
}

// URL error.
func urlError(err *url.Error) error {
	if err.Timeout() {
		return ErrTimeoutExceeded
	}

	if err.Err == io.EOF {
		return syscall.ECONNRESET
	}

	if err.Err == ErrMaxRedirectsExceeded {
		return ErrMaxRedirectsExceeded
	}

	if err, ok := err.Err.(*net.OpError); ok {
		return opError(err)
	}

	if _, ok := err.Err.(tls.RecordHeaderError); ok {
		return errors.New("invalid TLS record header")
	}

	if _, ok := err.Err.(x509.HostnameError); ok {
		return errors.New("SSL cert is not valid for this domain")
	}

	if _, ok := err.Err.(x509.UnknownAuthorityError); ok {
		return errors.New("SSL cert signed by unknown authority")
	}

	if err, ok := err.Err.(x509.CertificateInvalidError); ok {
		return certError(err)
	}

	if strings.Contains(err.Error(), "malformed HTTP") {
		return errors.New("malformed HTTP response")
	}

	return errors.New(strings.Replace(err.Err.Error(), "net/http: ", "", 1))
}

// Certificate error.
func certError(err x509.CertificateInvalidError) error {
	switch err.Reason {
	case x509.NotAuthorizedToSign:
		return errors.New("SSL cert is not authorized to sign others")
	case x509.Expired:
		return errors.New("SSL cert has expired")
	case x509.CANotAuthorizedForThisName:
		return errors.New("SSL a root or intermediate cert is not authorized to sign in this domain")
	case x509.TooManyIntermediates:
		return errors.New("SSL too many intermediates for path length constraint")
	case x509.IncompatibleUsage:
		return errors.New("SSL certificate specifies an incompatible key usage")
	case x509.NameMismatch:
		return errors.New("SSL issuer name does not match subject from issuing certificate")
	}

	return errors.New(strings.Replace(err.Error(), "x509: ", "SSL ", 1))
}

// Op error.
func opError(err *net.OpError) error {
	if err, ok := err.Err.(*os.SyscallError); ok {
		return syscallError(err)
	}

	if err, ok := err.Err.(*net.DNSError); ok {
		return dnsError(err)
	}

	return err
}

// Syscall error.
func syscallError(err *os.SyscallError) error {
	if err, ok := err.Err.(syscall.Errno); ok {
		return err
	}

	return err
}

// DNS error.
func dnsError(err *net.DNSError) error {
	return errors.New(err.Err)
}
