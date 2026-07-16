package phonevalidate

import (
	"fmt"
	"strings"
	"time"

	"github.com/nyaruka/phonenumbers"
)

// LocalTool identifies the offline engine and the pinned version this module's
// local scanner is built on. This is the maintained MIT Go port of Google's
// libphonenumber — the same engine PhoneInfoga's `local` scanner wraps. We build
// on it directly (rather than importing PhoneInfoga, which is GPL-3.0) so this
// MIT module incurs no copyleft obligation; see README.
const LocalTool = "nyaruka/phonenumbers@v1.5.0 (libphonenumber; PhoneInfoga local-scanner engine)"

// carrierLang is the language for offline carrier-name lookups.
const carrierLang = "en"

// LocalResult is the "local" sub-result of the phone_validate key: the offline,
// zero-API-key signals derived from the number itself. Status is "unknown" if
// the number could not be parsed at all.
type LocalResult struct {
	Status      string `json:"status"` // "ok" if the number parsed; else "unknown"
	FormatValid bool   `json:"format_valid"`
	IsValid     bool   `json:"is_valid_number"`
	LineType    string `json:"line_type"`
	Carrier     string `json:"carrier,omitempty"` // offline carrier; often empty (esp. number-portability regions like US)
	Country     string `json:"country"`           // ISO 3166-1 alpha-2 region, or "unknown"
	E164        string `json:"e164,omitempty"`
	National    string `json:"national,omitempty"`
	CountryCode int32  `json:"country_code,omitempty"`
	CheckedAt   string `json:"checked_at"`
	SourceTool  string `json:"source_tool"`
	Error       string `json:"error,omitempty"`
}

// lineTypeNames maps libphonenumber's PhoneNumberType enum to stable, lower-case
// strings for the JSON contract (the enum has no String() method). Keyed by the
// package's named constants so the mapping stays correct if the underlying
// numeric values ever change.
var lineTypeNames = map[phonenumbers.PhoneNumberType]string{
	phonenumbers.FIXED_LINE:           "fixed_line",
	phonenumbers.MOBILE:               "mobile",
	phonenumbers.FIXED_LINE_OR_MOBILE: "fixed_line_or_mobile",
	phonenumbers.TOLL_FREE:            "toll_free",
	phonenumbers.PREMIUM_RATE:         "premium_rate",
	phonenumbers.SHARED_COST:          "shared_cost",
	phonenumbers.VOIP:                 "voip",
	phonenumbers.PERSONAL_NUMBER:      "personal_number",
	phonenumbers.PAGER:                "pager",
	phonenumbers.UAN:                  "uan",
	phonenumbers.VOICEMAIL:            "voicemail",
	phonenumbers.UNKNOWN:              unknown,
}

// runLocal parses phone with libphonenumber and fills a LocalResult. It never
// panics the pipeline: an empty/unparseable number yields Status "unknown" with
// an Error note. It assumes an international/E.164-style number (the pipeline's
// ingest contract); a bare national number with no country code cannot be
// resolved offline and degrades to "unknown".
func runLocal(phone string, now time.Time) LocalResult {
	res := LocalResult{
		Status:     unknown,
		LineType:   unknown,
		Country:    unknown,
		CheckedAt:  now.Format(time.RFC3339),
		SourceTool: LocalTool,
	}

	e164ish := normalizePhone(phone)
	if e164ish == "" {
		if strings.TrimSpace(phone) == "" {
			res.Error = "no phone field present on lead record"
		} else {
			res.Error = fmt.Sprintf("phone field contains no digits: %q", phone)
		}
		return res
	}

	// Region hint is empty: the number is expected to carry its own country code
	// (leading "+"), matching the ingest contract. libphonenumber then derives
	// the region from the calling code.
	num, err := phonenumbers.Parse(e164ish, "")
	if err != nil {
		res.Error = "could not parse phone number: " + err.Error()
		return res
	}

	res.Status = "ok"
	res.FormatValid = phonenumbers.IsPossibleNumber(num)
	res.IsValid = phonenumbers.IsValidNumber(num)
	res.LineType = lineTypeName(phonenumbers.GetNumberType(num))
	res.E164 = phonenumbers.Format(num, phonenumbers.E164)
	res.National = phonenumbers.Format(num, phonenumbers.NATIONAL)
	res.CountryCode = num.GetCountryCode()

	if region := phonenumbers.GetRegionCodeForNumber(num); region != "" {
		res.Country = region
	}
	// Offline carrier lookup only returns a name for mobile-type numbers in
	// regions without number portability; empty is normal and left as such.
	if name, cErr := phonenumbers.GetCarrierForNumber(num, carrierLang); cErr == nil && name != "" {
		res.Carrier = name
	}
	return res
}

func lineTypeName(t phonenumbers.PhoneNumberType) string {
	if name, ok := lineTypeNames[t]; ok {
		return name
	}
	return unknown
}

// normalizePhone reduces a raw phone field to an E.164-style candidate: it keeps
// digits only and re-prepends a single leading "+", mirroring how PhoneInfoga's
// local scanner normalizes input before parsing. Punctuation, spaces, and a
// leading "00" international prefix collapse to a "+<digits>" string that
// libphonenumber can parse. Returns "" for empty/no-digit input.
func normalizePhone(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if digits == "" {
		return ""
	}
	return "+" + digits
}

// redact masks the middle of a phone string for audit logs so raw PII does not
// leak into log sinks, while keeping enough to correlate a run. Fewer than 5
// retained-ish digits collapse to a fully masked token.
func redact(phone string) string {
	var digits []rune
	var plus bool
	for _, r := range strings.TrimSpace(phone) {
		if r == '+' && len(digits) == 0 {
			plus = true
		}
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	prefix := ""
	if plus {
		prefix = "+"
	}
	if len(digits) == 0 {
		return ""
	}
	if len(digits) <= 4 {
		return prefix + strings.Repeat("*", len(digits))
	}
	head := string(digits[:2])
	tail := string(digits[len(digits)-2:])
	return prefix + head + strings.Repeat("*", len(digits)-4) + tail
}
