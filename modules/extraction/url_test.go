package extraction

import (
	"net"
	"net/url"
	"strings"
	"testing"
)

func TestValidateURL_AcceptsPublicHTTPS(t *testing.T) {
	u, err := validateURL("https://example.com/path?q=1", fakePublicResolve)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Scheme != "https" {
		t.Errorf("scheme = %q", u.Scheme)
	}
	if u.Host != "example.com" {
		t.Errorf("host = %q", u.Host)
	}
}

func TestValidateURL_AcceptsPublicHTTP(t *testing.T) {
	u, err := validateURL("http://example.com/", fakePublicResolve)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Scheme != "http" {
		t.Errorf("scheme = %q", u.Scheme)
	}
}

func TestValidateURL_RejectNonHTTP(t *testing.T) {
	for _, scheme := range []string{"ftp", "file", "data", "javascript"} {
		_, err := validateURL(scheme+"://example.com/", fakePublicResolve)
		if err == nil {
			t.Errorf("scheme %q should be rejected", scheme)
		}
	}
}

func TestValidateURL_RejectEmptyURL(t *testing.T) {
	_, err := validateURL("", fakePublicResolve)
	if err == nil {
		t.Fatal("empty URL should be rejected")
	}
}

func TestValidateURL_RejectUserinfo(t *testing.T) {
	_, err := validateURL("https://user:pass@example.com/", fakePublicResolve)
	if err == nil {
		t.Fatal("credentialed URL should be rejected")
	}
	if !strings.Contains(err.Error(), "credentials") {
		t.Errorf("error = %q", err)
	}
}

func TestValidateURL_RejectIPLiteral(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "93.184.216.34"} {
		_, err := validateURL("https://"+host+"/", fakePublicResolve)
		if err == nil {
			t.Errorf("IP-literal %q should be rejected", host)
		}
		if !strings.Contains(err.Error(), "IP-literal") {
			t.Errorf("error for %q does not mention IP-literal: %q", host, err)
		}
	}
}

func TestValidateURL_RejectPrivateDNS(t *testing.T) {
	_, err := validateURL("https://example.com/", fakePrivateResolve)
	if err == nil {
		t.Fatal("hostname resolving to private IP should be rejected")
	}
	if !strings.Contains(err.Error(), "forbidden IP") {
		t.Errorf("error = %q", err)
	}
}

func TestValidateURL_RejectNonStandardPort(t *testing.T) {
	for _, port := range []string{"8080", "3000", "22", "8000"} {
		_, err := validateURL("https://example.com:"+port+"/", fakePublicResolve)
		if err == nil {
			t.Errorf("port %q should be rejected", port)
		}
	}
}

func TestValidateURL_AcceptStandardPorts(t *testing.T) {
	for _, port := range []string{"80", "443"} {
		u, err := validateURL("https://example.com:"+port+"/", fakePublicResolve)
		if err != nil {
			t.Errorf("port %q should be accepted: %v", port, err)
		}
		if u.Port() != port {
			t.Errorf("port = %q", u.Port())
		}
	}
}

func TestValidateURL_RejectMissingHost(t *testing.T) {
	_, err := validateURL("https:///", fakePublicResolve)
	if err == nil {
		t.Fatal("missing host should be rejected")
	}
}

func TestIsForbiddenIP(t *testing.T) {
	cases := []struct {
		ip     string
		forbid bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.0.1", true},
		{"169.254.169.254", true},
		{"100.64.0.1", true},
		{"100.127.255.254", true},
		{"8.8.8.8", false},
		{"93.184.216.34", false},
		{"::1", true},
		{"fe80::1", true},
		{"fc00::1", true},
		{"ff02::1", true},
		{"0.0.0.0", true},
		{"::", true},
	}
	for _, tc := range cases {
		ip := net.ParseIP(tc.ip)
		if ip == nil {
			t.Fatalf("invalid test IP %q", tc.ip)
		}
		got := isForbiddenIP(ip)
		if got != tc.forbid {
			t.Errorf("isForbiddenIP(%q) = %v, want %v", tc.ip, got, tc.forbid)
		}
	}
}

func TestSanitizeURLForAudit_RedactsQueryAndStripsUserinfo(t *testing.T) {
	u, err := validateURL("https://example.com/path?token=abc&id=123", fakePublicResolve)
	if err != nil {
		t.Fatalf("validateURL: %v", err)
	}
	s := sanitizeURLForAudit(u)
	if strings.Contains(s, "token=abc") || strings.Contains(s, "id=123") {
		t.Errorf("audit URL leaked query values: %q", s)
	}
	if !strings.Contains(s, "[redacted]") && !strings.Contains(s, "%5Bredacted%5D") {
		t.Errorf("audit URL did not redact query: %q", s)
	}

	// Direct parse to test sanitization of userinfo.
	u2, err := url.Parse("https://user:pass@example.com/?q=1")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	s2 := sanitizeURLForAudit(u2)
	if strings.Contains(s2, "user:pass") || strings.Contains(s2, "user:") {
		t.Errorf("audit URL leaked credentials: %q", s2)
	}
	if strings.Contains(s2, "q=1") {
		t.Errorf("audit URL leaked query value: %q", s2)
	}
}
