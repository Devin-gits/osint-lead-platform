package domainintel

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/likexian/whois"
)

// WebCheckTool identifies this sub-tool and its provenance. This module does not
// fork or run lissy93/web-check's full Node app (see README §"web-check
// integration"); it reimplements the specific domain-infra checks web-check
// exposes at /api/dns, /api/ssl, HTTP and /api/whois — the "is this an established,
// real business domain?" signals — using the Go standard library. The version
// string names the upstream project the checks are modeled on.
const WebCheckTool = "web-check-lite (reimpl. of lissy93/web-check@2.1.10 dns/tls/http/whois checks)"

// DNSResult mirrors the record-type shape of web-check's api/dns.js JSON output
// (A/AAAA/CNAME/MX/NS/TXT), restricted to the record types this module consumes.
type DNSResult struct {
	A     []string `json:"a"`
	AAAA  []string `json:"aaaa"`
	CNAME []string `json:"cname"`
	MX    []string `json:"mx"`
	NS    []string `json:"ns"`
	TXT   []string `json:"txt"`
}

// SSLResult captures the leaf-certificate facts web-check's api/ssl surfaces:
// whether a valid chain was presented, who issued it, and its validity window —
// the signal for "this domain terminates real, currently-valid TLS".
type SSLResult struct {
	Valid           bool     `json:"valid"`
	Issuer          string   `json:"issuer,omitempty"`
	Subject         string   `json:"subject,omitempty"`
	NotBefore       string   `json:"not_before,omitempty"`
	NotAfter        string   `json:"not_after,omitempty"`
	DaysUntilExpiry int      `json:"days_until_expiry"`
	Protocol        string   `json:"protocol,omitempty"`
	SANs            []string `json:"sans,omitempty"`
	Error           string   `json:"error,omitempty"`
}

type HTTPResult struct {
	StatusCode int               `json:"status_code"`
	Server     string            `json:"server,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Error      string            `json:"error,omitempty"`
}

// WhoisResult carries the domain-age signal web-check's api/whois surfaces:
// registrar and creation date, plus the derived age in days. DomainAgeDays is
// the lead-quality signal (very young domains are higher-risk).
type WhoisResult struct {
	Registrar     string `json:"registrar,omitempty"`
	CreatedDate   string `json:"created_date,omitempty"`
	DomainAgeDays int    `json:"domain_age_days"`
	Error         string `json:"error,omitempty"`
}

// WebCheckResult is the "web_check" sub-result of the domain_intel key. It
// answers the establishment/legitimacy question; Status is "unknown" if the
// whole sub-tool failed, but individual checks (dns/tls/http/whois) degrade
// independently and carry their own error notes.
type WebCheckResult struct {
	Status     string       `json:"status"` // "ok" if at least DNS resolved; else "unknown"
	Resolvable bool         `json:"resolvable"`
	DNS        *DNSResult   `json:"dns,omitempty"`
	SSL        *SSLResult   `json:"ssl,omitempty"`
	HTTP       *HTTPResult  `json:"http,omitempty"`
	Whois      *WhoisResult `json:"whois,omitempty"`
	CheckedAt  string       `json:"checked_at"`
	SourceTool string       `json:"source_tool"`
	Error      string       `json:"error,omitempty"`
}

// runWebCheck performs the DNS, TLS, HTTP and WHOIS checks against domain, each
// bounded by the shared timeout and each degrading independently so one failing
// check never blocks the others. It never panics: a total failure returns a
// Status "unknown" result rather than an error.
func runWebCheck(ctx context.Context, domain string, timeout time.Duration) WebCheckResult {
	now := time.Now().UTC()
	res := WebCheckResult{
		Status:     "unknown",
		CheckedAt:  now.Format(time.RFC3339),
		SourceTool: WebCheckTool,
	}

	domain = normalizeDomain(domain)
	if domain == "" {
		res.Error = "no domain field present on lead record"
		return res
	}

	dns := lookupDNS(ctx, domain, timeout)
	res.DNS = &dns
	res.Resolvable = len(dns.A) > 0 || len(dns.AAAA) > 0
	if res.Resolvable {
		// DNS resolving is the minimum bar for the sub-tool to count as "ok";
		// SSL/WHOIS are best-effort enrichment on top.
		res.Status = "ok"
	}

	ssl := inspectSSL(domain, timeout)
	res.SSL = &ssl

	httpResult := inspectHTTP(ctx, domain, timeout)
	res.HTTP = &httpResult

	whois := lookupWhois(domain, timeout)
	res.Whois = &whois

	if !res.Resolvable {
		res.Error = "domain did not resolve to any A/AAAA record"
	}
	return res
}

// normalizeDomain strips scheme, path, port and userinfo so callers may pass a
// bare domain or a URL (web-check accepts a URL; the pipeline passes a bare
// domain). Mirrors the intent of web-check's api/_common/parse-target.js.
func normalizeDomain(d string) string {
	d = strings.TrimSpace(strings.ToLower(d))
	if d == "" {
		return ""
	}
	if i := strings.Index(d, "://"); i >= 0 {
		d = d[i+3:]
	}
	if i := strings.Index(d, "@"); i >= 0 {
		d = d[i+1:]
	}
	if i := strings.IndexAny(d, "/?#"); i >= 0 {
		d = d[:i]
	}
	if h, _, err := net.SplitHostPort(d); err == nil {
		d = h
	}
	return strings.TrimSuffix(d, ".")
}

func lookupDNS(ctx context.Context, domain string, timeout time.Duration) DNSResult {
	c, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var r net.Resolver
	out := DNSResult{A: []string{}, AAAA: []string{}, CNAME: []string{}, MX: []string{}, NS: []string{}, TXT: []string{}}

	if ips, err := r.LookupIP(c, "ip4", domain); err == nil {
		for _, ip := range ips {
			out.A = append(out.A, ip.String())
		}
	}
	if ips, err := r.LookupIP(c, "ip6", domain); err == nil {
		for _, ip := range ips {
			out.AAAA = append(out.AAAA, ip.String())
		}
	}
	if cname, err := r.LookupCNAME(c, domain); err == nil {
		cname = strings.TrimSuffix(cname, ".")
		if cname != "" && cname != strings.TrimSuffix(domain, ".") {
			out.CNAME = append(out.CNAME, cname)
		}
	}
	if mxs, err := r.LookupMX(c, domain); err == nil {
		for _, mx := range mxs {
			out.MX = append(out.MX, strings.TrimSuffix(mx.Host, "."))
		}
	}
	if nss, err := r.LookupNS(c, domain); err == nil {
		for _, ns := range nss {
			out.NS = append(out.NS, strings.TrimSuffix(ns.Host, "."))
		}
	}
	if txts, err := r.LookupTXT(c, domain); err == nil {
		out.TXT = append(out.TXT, txts...)
	}
	return out
}

// inspectSSL dials the domain on 443 and reads the presented leaf certificate.
// A failed handshake (no TLS, expired/invalid chain, timeout) is reported via
// Valid=false + Error, not a panic.
func inspectSSL(domain string, timeout time.Duration) SSLResult {
	out := SSLResult{}
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(domain, "443"),
		&tls.Config{ServerName: domain, MinVersion: tls.VersionTLS12})
	if err != nil {
		out.Error = fmt.Sprintf("tls handshake failed: %v", err)
		return out
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		out.Error = "no peer certificate presented"
		return out
	}
	state := conn.ConnectionState()
	leaf := certs[0]
	out.Valid = true
	out.Issuer = leaf.Issuer.String()
	out.Subject = leaf.Subject.String()
	out.NotBefore = leaf.NotBefore.UTC().Format(time.RFC3339)
	out.NotAfter = leaf.NotAfter.UTC().Format(time.RFC3339)
	out.DaysUntilExpiry = int(time.Until(leaf.NotAfter).Hours() / 24)
	out.Protocol = tlsVersionName(state.Version)
	out.SANs = append([]string(nil), leaf.DNSNames...)
	return out
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS10:
		return "TLS 1.0"
	default:
		return "unknown"
	}
}

func inspectHTTP(ctx context.Context, domain string, timeout time.Duration) HTTPResult {
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{Timeout: timeout}
	var lastErr error

	for _, scheme := range []string{"https", "http"} {
		req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, scheme+"://"+domain, nil)
		if err != nil {
			return HTTPResult{Error: fmt.Sprintf("http request creation failed: %v", err)}
		}
		req.Header.Set("User-Agent", "domain-intel/1.0")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()

		headers := make(map[string]string)
		for _, key := range []string{"Content-Type", "X-Powered-By"} {
			if value := resp.Header.Get(key); value != "" {
				headers[key] = value
			}
		}
		return HTTPResult{
			StatusCode: resp.StatusCode,
			Server:     resp.Header.Get("Server"),
			Headers:    headers,
		}
	}

	return HTTPResult{Error: fmt.Sprintf("http request failed: %v", lastErr)}
}

var (
	reWhoisCreated   = regexp.MustCompile(`(?i)^\s*(?:Creation Date|Created On|Created Date|created|Registered on|Domain Registration Date|Registration Time):\s*(.+?)\s*$`)
	reWhoisRegistrar = regexp.MustCompile(`(?i)^\s*Registrar:\s*(.+?)\s*$`)
)

// lookupWhois queries the domain through the Apache-2.0 likexian/whois client,
// which follows registry referrals and supports bounded network timeouts. Local
// parsing keeps the module's stable registrar/domain-age output schema.
func lookupWhois(domain string, timeout time.Duration) WhoisResult {
	out := WhoisResult{}
	client := whois.NewClient().SetTimeout(timeout)
	response, err := client.Whois(domain)
	if err != nil {
		out.Error = fmt.Sprintf("whois query failed: %v", err)
		return out
	}

	for _, line := range strings.Split(response, "\n") {
		if out.CreatedDate == "" {
			if m := reWhoisCreated.FindStringSubmatch(line); m != nil {
				out.CreatedDate = strings.TrimSpace(m[1])
			}
		}
		if out.Registrar == "" {
			if m := reWhoisRegistrar.FindStringSubmatch(line); m != nil {
				out.Registrar = strings.TrimSpace(m[1])
			}
		}
	}

	if out.CreatedDate != "" {
		if created, ok := parseWhoisDate(out.CreatedDate); ok {
			out.DomainAgeDays = int(time.Since(created).Hours() / 24)
		}
	}
	if out.CreatedDate == "" && out.Registrar == "" {
		out.Error = "whois response contained no creation-date or registrar field"
	}
	return out
}

// whoisDateLayouts are the creation-date formats seen across common registries.
var whoisDateLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05.000Z",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"02-Jan-2006",
	"2006.01.02",
	"20060102",
}

func parseWhoisDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, layout := range whoisDateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
