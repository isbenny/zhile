package web

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Guest-supplied artwork must never forward the administrator's platform
// cookies or reach loopback/private networks, including through redirects/DNS.
func publicMediaIP(ip net.IP) bool {
	v4 := ip.To4()
	return ip != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !(v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127)
}
func publicImageURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || (u.Port() != "" && u.Port() != "80" && u.Port() != "443") {
		return errors.New("invalid image URL")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && !publicMediaIP(ip) {
		return errors.New("private image address")
	}
	return nil
}
func memberImageTransport() *http.Transport {
	return &http.Transport{TLSHandshakeTimeout: 8 * time.Second, ResponseHeaderTimeout: 10 * time.Second, DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if !publicMediaIP(ip.IP) {
				return nil, errors.New("private image address")
			}
		}
		for _, ip := range ips {
			conn, e := (&net.Dialer{Timeout: 8 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
			if e == nil {
				return conn, nil
			}
			err = e
		}
		if err == nil {
			err = errors.New("no image address")
		}
		return nil, err
	}}
}
func fetchMemberImage(ctx context.Context, raw string) ([]byte, string, error) {
	if err := publicImageURL(raw); err != nil {
		return nil, "", err
	}
	transport := memberImageTransport()
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 20 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 4 {
			return errors.New("too many redirects")
		}
		return publicImageURL(req.URL.String())
	}}
	req, err := http.NewRequestWithContext(ctx, "GET", raw, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, "", errors.New("image unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, (10<<20)+1))
	if err != nil || len(data) > 10<<20 {
		return nil, "", errors.New("image too large")
	}
	mime := http.DetectContentType(data)
	if !strings.HasPrefix(mime, "image/") {
		return nil, "", errors.New("invalid image content")
	}
	return data, mime, nil
}
