package socialfootprint

import (
	"strings"
)

// handleCandidate is one derived username to spot-check, with a record of how it
// was derived (surfaced in the output and used for auditing/debuggability).
type handleCandidate struct {
	handle string
	origin string
}

// origin labels — how a candidate handle was derived.
const (
	originDirect       = "direct"
	originEmailLocal   = "email-local-part"
	originEmailVariant = "email-variant"
	originDomainIntel  = "domain-intel-harvester"
)

// deriveHandles turns a (possibly enriched) lead record into an ordered,
// deduplicated list of candidate handles to check. Primary source is the email
// local-part plus a couple of cheap variants; secondary/best-effort source is an
// already-present domain_intel.harvester sub-object. Candidates are returned in
// priority order (best first); the caller caps the count at MaxHandles.
//
// This is the "handle dependency" resolved internally: Maigret needs a username,
// the raw lead has none, so we synthesize candidates here rather than requiring a
// separate upstream module.
func deriveHandles(lead map[string]interface{}) []handleCandidate {
	var out []handleCandidate
	seen := map[string]bool{}

	add := func(h, origin string) {
		h = normalizeHandle(h)
		if h == "" || seen[h] {
			return
		}
		seen[h] = true
		out = append(out, handleCandidate{handle: h, origin: origin})
	}

	// Optional direct username/handle from a CLI flag or orchestrator; takes
	// precedence over email-derived candidates because it is explicitly supplied.
	if handle, _ := lead["username"].(string); handle != "" {
		add(handle, originDirect)
	}

	// Primary: the email local-part and 2-3 sane variants.
	if email, _ := lead["email"].(string); email != "" {
		local := emailLocalPart(email)
		if local != "" {
			add(local, originEmailLocal)
			for _, v := range emailVariants(local) {
				add(v, originEmailVariant)
			}
		}
	}

	// Secondary (best-effort): an already-enriched domain_intel.harvester
	// sub-object. Only used when present — the module does not require it and
	// does not call domain-intel itself.
	for _, h := range harvesterHandles(lead) {
		add(h, originDomainIntel)
	}

	return out
}

// emailLocalPart returns the part before '@', with any "+tag" suffix stripped
// (jane+news@x -> jane). Returns "" if the email has no local-part.
func emailLocalPart(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	at := strings.Index(email, "@")
	if at <= 0 {
		return ""
	}
	local := email[:at]
	if plus := strings.Index(local, "+"); plus >= 0 {
		local = local[:plus]
	}
	return local
}

// emailVariants derives up to two extra cheap handle guesses from a dotted
// local-part: the dots-removed form (jane.smith -> janesmith) and the
// first-initial+last form (jane.smith -> jsmith). Non-dotted locals yield no
// variants. This is deliberately conservative (2 variants max) per the task's
// "don't go overboard".
func emailVariants(local string) []string {
	parts := strings.FieldsFunc(local, func(r rune) bool {
		return r == '.' || r == '_' || r == '-'
	})
	if len(parts) < 2 {
		return nil
	}
	var out []string
	// dots/separators removed
	out = append(out, strings.Join(parts, ""))
	// first initial + last token
	first := parts[0]
	last := parts[len(parts)-1]
	if first != "" && last != "" {
		out = append(out, first[:1]+last)
	}
	return out
}

// infraLabels are hostname labels that are never a person's handle; excluded
// when mining host fragments from domain_intel.harvester.
var infraLabels = map[string]bool{
	"www": true, "mail": true, "smtp": true, "imap": true, "pop": true,
	"ns": true, "ns1": true, "ns2": true, "mx": true, "ftp": true,
	"cpanel": true, "webmail": true, "autodiscover": true, "vpn": true,
	"api": true, "cdn": true, "static": true, "assets": true, "img": true,
	"blog": true, "dev": true, "test": true, "staging": true, "portal": true,
	"remote": true, "server": true, "gateway": true, "proxy": true, "mailer": true,
	"email": true, "secure": true, "login": true, "app": true, "web": true,
}

// harvesterHandles mines best-effort handle candidates from an already-present
// domain_intel.harvester sub-object (the shape domain-intel emits). It prefers
// email local-parts discovered by theHarvester (most handle-like), then falls
// back to the leading label of discovered hostnames, skipping infrastructure
// labels. Returns at most a handful — the overall MaxHandles cap applies after.
func harvesterHandles(lead map[string]interface{}) []string {
	di, ok := lead["domain_intel"].(map[string]interface{})
	if !ok {
		return nil
	}
	h, ok := di["harvester"].(map[string]interface{})
	if !ok {
		return nil
	}

	var out []string

	// 1. Local-parts of harvester-discovered emails.
	if emails, ok := h["emails"].([]interface{}); ok {
		for _, e := range emails {
			if s, ok := e.(string); ok {
				if lp := emailLocalPart(s); lp != "" {
					out = append(out, lp)
				}
			}
		}
	}

	// 2. Leading label of discovered hostnames (the "hostname fragment"),
	//    excluding infra labels like www/mail/ns.
	if hosts, ok := h["hosts"].([]interface{}); ok {
		for _, hv := range hosts {
			hm, ok := hv.(map[string]interface{})
			if !ok {
				continue
			}
			host, _ := hm["host"].(string)
			if frag := leadingLabel(host); frag != "" && !infraLabels[frag] {
				out = append(out, frag)
			}
		}
	}

	return out
}

// leadingLabel returns the first dot-separated label of a hostname, lowercased.
func leadingLabel(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}
	if dot := strings.Index(host, "."); dot > 0 {
		return host[:dot]
	}
	return host
}

// normalizeHandle lowercases, trims, and validates a candidate handle. It strips
// common noise such as a leading '@', URLs, and trailing path/query fragments,
// then keeps only plausible username characters (letters, digits, '.', '_', '-')
// and rejects handles that are too short or contain no letter.
func normalizeHandle(h string) string {
	h = strings.TrimSpace(strings.ToLower(h))
	if h == "" {
		return ""
	}

	// Strip a leading '@' (common copy-paste noise).
	h = strings.TrimPrefix(h, "@")

	// Strip common URL prefixes and path/query fragments.
	h = stripURLPrefix(h)

	if h == "" {
		return ""
	}

	var b strings.Builder
	hasLetter := false
	for _, r := range h {
		switch {
		case r >= 'a' && r <= 'z':
			hasLetter = true
			b.WriteRune(r)
		case r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
			// other characters are dropped
		}
	}
	clean := strings.Trim(b.String(), "._-")
	if len(clean) < 2 || !hasLetter {
		return ""
	}
	return clean
}

// stripURLPrefix removes http(s):// and www. prefixes, and returns the final
// path segment (the handle) if the string looks like a profile URL. It also
// strips a trailing query string. '@' inside the path is left to normalizeHandle
// (it drops the '@' and keeps surrounding letters), except for a leading '@'
// which normalizeHandle strips explicitly.
func stripURLPrefix(h string) string {
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimPrefix(h, "http://")
	h = strings.TrimPrefix(h, "www.")

	if i := strings.Index(h, "?"); i >= 0 {
		h = h[:i]
	}
	if i := strings.LastIndex(h, "/"); i >= 0 {
		h = h[i+1:]
	}
	return strings.TrimSpace(h)
}
