package oidc

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

const maxOutboundRedirects = 5

var errBlockedAddress = errors.New("oidc: refusing to connect to a non-public address")

// newFederationClient is the HTTP client for every outbound social and SSO
// call. It refuses private, link-local, shared, unspecified and multicast
// addresses at every dial and redirect, so admin-configured or user-started
// requests cannot reach the server's own network (cloud metadata, internal
// databases).
//
// The check runs on the resolved address, so DNS cannot hide one. Loopback is
// allowed only when the server itself runs on loopback (development); in
// production it is the admin listener.
func newFederationClient(timeout time.Duration, allowLoopback bool) *http.Client {
	dialer := &net.Dialer{Timeout: timeout}
	dialer.Control = func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		ip := net.ParseIP(host)
		if ip == nil || blockedAddress(ip) || ip.IsLoopback() && !allowLoopback {
			return errBlockedAddress
		}
		return nil
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{DialContext: dialer.DialContext},
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= maxOutboundRedirects {
				return errors.New("oidc: too many redirects")
			}
			return nil
		},
	}
}

// blockedAddress reports whether a resolved address is one no federation
// request may reach. Loopback is decided by newFederationClient. 100.64.0.0/10
// is shared address space, where some clouds keep their metadata service.
func blockedAddress(ip net.IP) bool {
	return ip.IsPrivate() ||
		sharedAddressSpace.Contains(ip) ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

var sharedAddressSpace = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

func onLoopback(address string) bool {
	parsed, err := url.Parse(address)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
