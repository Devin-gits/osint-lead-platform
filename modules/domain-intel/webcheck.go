package domainintel

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

// WebCheckTool identifies this sub-tool and its provenance. This module does not
// fork or run lissy93/web-check's full Node app (see README §"web-check
// integration"); it reimplements the specific domain-infra checks web-check
// exposes at /api/dns, /api/ssl and /api/whois — the "is this an established,
// real business domain?" signals — using the Go standard library. The version
// string names the upstream project the checks are modeled on.
const WebCheckTool = "web-check-lite (reimpl. of lissy93/web-check@2.1.10 dns/ssl/whois checks)"

// DNSResult mirrors the record-type shape of web-check's api/dns.js JSON output
// (A/AAAA/MX/NS/TXT), restricted to the record types this module consumes.
type DNSResult struct {
	A    []string `json:"a"`
	AAAA []string `json:"aaaa"`
	MX   []string `json:"mx"`
	NS   []string `json:"ns"`
	TXT  []string `json:"txt"`
}

// SSLResult captures the leaf-certificate facts web-check's api/ssl surfaces:
// whether a valid chain was presented, who issued it, and its validity window —
// the signal for "this domain terminates real, currently-valid TLS".
type SSLResult struct {
	Valid           bool   `json:"valid"`
	Issuer          string `json:"issuer,omitempty"`
	Subject         string `json:"subject,omitempty"`
	NotBefore       string `json:"not_before,omitempty"`
	NotAfter        string `json:"not_after,omitempty"`
	DaysUntilExpiry int    `json:"days_until_expiry"`
	Error           string `json:"error,omitempty"`
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
// whole sub-tool failed, but individual checks (dns/ssl/whois) degrade
// independently and carry their own error notes.
type WebCheckResult struct {
	Status     string       `json:"status"` // "ok" if at least DNS resolved; else "unknown"
	Resolvable bool         `json:"resolvable"`
	DNS        *DNSResult   `json:"dns,omitempty"`
	SSL        *SSLResult   `json:"ssl,omitempty"`
	Whois      *WhoisResult `json:"whois,omitempty"`
	CheckedAt  string       `json:"checked_at"`
	SourceTool string       `json:"source_tool"`
	Error      string       `json:"error,omitempty"`
}

// runWebCheck performs the DNS, SSL and WHOIS checks against domain, each
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
	out := DNSResult{A: []string{}, AAAA: []string{}, MX: []string{}, NS: []string{}, TXT: []string{}}

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
	leaf := certs[0]
	out.Valid = true
	out.Issuer = leaf.Issuer.String()
	out.Subject = leaf.Subject.String()
	out.NotBefore = leaf.NotBefore.UTC().Format(time.RFC3339)
	out.NotAfter = leaf.NotAfter.UTC().Format(time.RFC3339)
	out.DaysUntilExpiry = int(time.Until(leaf.NotAfter).Hours() / 24)
	return out
}

var (
	reWhoisRefer     = regexp.MustCompile(`(?i)^\s*(?:refer|whois):\s*(\S+)`)
	reWhoisCreated   = regexp.MustCompile(`(?i)^\s*(?:Creation Date|Created On|Created Date|created|Registered on|Domain Registration Date|Registration Time):\s*(.+?)\s*$`)
	reWhoisRegistrar = regexp.MustCompile(`(?i)^\s*Registrar:\s*(.+?)\s*$`)
)

// lookupWhois performs a two-step WHOIS query over TCP/43 using only the
// standard library: it asks whois.iana.org which server is authoritative for
// the domain's TLD, then queries that server for the domain and parses the
// creation date and registrar. This is the same referral chain a `whois`
// client follows; we implement it directly because Go has no stdlib WHOIS and
// the sandbox has no `whois` binary. Any failure degrades to an Error note.
func lookupWhois(domain string, timeout time.Duration) WhoisResult {
	out := WhoisResult{}

	tld := domain
	if i := strings.LastIndex(domain, "."); i >= 0 {
		tld = domain[i+1:]
	}

	ianaResp, err := whoisQuery("whois.iana.org", tld, timeout)
	if err != nil {
		out.Error = fmt.Sprintf("iana referral lookup failed: %v", err)
		return out
	}
	server := ""
	for _, line := range strings.Split(ianaResp, "\n") {
		if m := reWhoisRefer.FindStringSubmatch(line); m != nil {
			server = strings.TrimSpace(m[1])
			break
		}
	}
	if server == "" {
		out.Error = fmt.Sprintf("no referral whois server for TLD %q", tld)
		return out
	}

	resp, err := whoisQuery(server, domain, timeout)
	if err != nil {
		out.Error = fmt.Sprintf("whois query to %s failed: %v", server, err)
		return out
	}

	for _, line := range strings.Split(resp, "\n") {
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

// whoisQuery opens a TCP/43 connection, sends the query terminated by CRLF (the
// WHOIS protocol, RFC 3912), and returns the full response, bounded by timeout.
func whoisQuery(server, query string, timeout time.Duration) (string, error) {
	d := &net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", net.JoinHostPort(server, "43"))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))
	if _, err := conn.Write([]byte(query + "\r\n")); err != nil {
		return "", err
	}
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	return sb.String(), nil
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
