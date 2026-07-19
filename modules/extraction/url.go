package extraction

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// URL validation constants.
const (
	MaxRedirects = 5
	MaxBodyBytes = 2 * 1024 * 1024
)

// LimitsApplied is a human-readable summary of the guardrails enforced for
// every extraction. It is included in audit records and result metadata.
const LimitsApplied = "max_body=2MB,max_markdown=100KB,timeout=45s,max_redirects=5"

// validateTargetURL parses raw and enforces the extraction SSRF policy using
// the Extractor's configured resolver. See validateURL for the full policy.
func (e *Extractor) validateTargetURL(raw string) (*url.URL, error) {
	return validateURL(raw, e.resolve)
}

// validateURL parses raw and enforces the extraction SSRF policy:
//   - http or https only
//   - no userinfo (credentials) in the URL
//   - a non-empty hostname
//   - no IP-literal hostnames by default
//   - only standard ports (80, 443) allowed by default
//   - DNS resolution must not point to loopback, link-local, RFC1918, CGNAT,
//     unique-local IPv6, multicast, unspecified, or cloud metadata IPs.
//
// It returns the parsed *url.URL so callers can fetch from u.String() while
// using sanitizeURLForAudit(u) for logging/audit. The resolve hook is injectable
// for tests and for the Firecrawl adapter's redirect checker.
func validateURL(raw string, resolve func(string) ([]net.IP, error)) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty URL")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	if u.User != nil {
		return nil, fmt.Errorf("URL must not contain credentials")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("URL scheme %q not allowed; only http and https", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("URL has no host")
	}

	// Reject IP-literal hostnames by default. Public websites should use DNS
	// names; fetching an IP address directly bypasses DNS-level controls and is
	// a common SSRF pattern.
	if ip := net.ParseIP(host); ip != nil {
		if isForbiddenIP(ip) {
			return nil, fmt.Errorf("URL host %q is a forbidden IP address; IP-literal URLs are not allowed", host)
		}
		return nil, fmt.Errorf("IP-literal URLs are not allowed")
	}

	// Port restriction: only 80 and 443 by default.
	if port := u.Port(); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid port %q", port)
		}
		if p != 80 && p != 443 {
			return nil, fmt.Errorf("non-standard port %d not allowed", p)
		}
	}

	// Resolve the hostname and validate every returned IP. This catches DNS
	// rebinding at lookup time; we revalidate at connection time where the
	// HTTP client allows it (Firecrawl adapter CheckRedirect).
	if resolve == nil {
		resolve = net.LookupIP
	}
	ips, err := resolve(host)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup for %q: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no DNS records for %q", host)
	}
	for _, ip := range ips {
		if isForbiddenIP(ip) {
			return nil, fmt.Errorf("URL %q resolves to forbidden IP %s", raw, ip)
		}
	}

	return u, nil
}

// isForbiddenIP reports whether ip is a loopback, link-local, private,
// carrier-grade NAT, unique-local IPv6, multicast, unspecified, or known
// cloud-metadata address.
func isForbiddenIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	// Go's IsPrivate covers RFC1918 IPv4 and RFC4193 IPv6 unique-local.
	// IsLoopback, IsLinkLocalUnicast, IsMulticast, and IsUnspecified cover
	// the other clearly internal/reserved ranges.
	if ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() ||
		ip.IsPrivate() {
		return true
	}

	// Carrier-grade NAT (RFC 6598): 100.64.0.0/10.
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
	}

	// Cloud metadata endpoint: 169.254.169.254/32.
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 && ip4[2] == 169 && ip4[3] == 254 {
			return true
		}
	}

	return false
}

// sanitizeURLForResult returns a URL string safe to store in a result: user
// credentials and fragment identifiers are stripped, but the query string is
// preserved so the landing page identity remains intact.
func sanitizeURLForResult(u *url.URL) string {
	if u == nil {
		return ""
	}
	out := *u
	out.User = nil
	out.Fragment = ""
	out.RawFragment = ""
	return out.String()
}

// sanitizeURLForAudit returns a URL string suitable for audit logs: user
// credentials and fragment identifiers are stripped, and query parameter values
// are replaced with "[redacted]" so PII/session tokens do not leak into the
// audit trail.
func sanitizeURLForAudit(u *url.URL) string {
	if u == nil {
		return ""
	}

	out := *u
	out.User = nil
	out.Fragment = ""
	out.RawFragment = ""

	if out.RawQuery != "" {
		q := out.Query()
		for k := range q {
			q.Set(k, "[redacted]")
		}
		out.RawQuery = q.Encode()
	}

	return out.String()
}

// sanitizeURLStringResult parses raw and sanitizes it for result storage.
func sanitizeURLStringResult(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return sanitizeURLForResult(u)
}

// sanitizeURLStringAudit parses raw and sanitizes it for audit logging. If
// parsing fails, the original string is returned unchanged (the audit must still
// record the target even when validation fails).
func sanitizeURLStringAudit(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return sanitizeURLForAudit(u)
}
