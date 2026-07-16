package phonevalidate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// NumverifyTool identifies the optional carrier-lookup source. numverify is an
// APILayer product; per the Stage 1 decision it is treated as a thin, swappable,
// OPTIONAL dependency — not required for the module to function.
const NumverifyTool = "numverify (apilayer) /validate"

// Environment configuration for the optional numverify path.
const (
	// APIKeyEnv, when unset/empty, makes the numverify sub-result "skipped".
	APIKeyEnv = "NUMVERIFY_API_KEY"
	// BaseURLEnv overrides the numverify endpoint (used for testing against a
	// local stub; also lets a deployment switch to https/paid or a successor
	// provider without a code change).
	BaseURLEnv = "NUMVERIFY_BASE_URL"
)

// defaultNumverifyBaseURL is numverify's documented validation endpoint. The
// free tier historically serves plain HTTP on apilayer.net; a paid plan / the
// apilayer.com marketplace host supports HTTPS. Override with BaseURLEnv.
const defaultNumverifyBaseURL = "https://apilayer.net/api/validate"

// Numverify sub-result statuses.
const (
	StatusOK      = "ok"
	StatusSkipped = "skipped" // no API key configured — NOT a failure
	StatusUnknown = unknown   // attempted but failed (network/HTTP/API error)
)

// NumverifyResult is the "numverify" sub-result of the phone_validate key. When
// no API key is set it is Status "skipped" and the module still returns full
// local-scanner results. Fields mirror numverify's documented /validate JSON.
type NumverifyResult struct {
	Status      string `json:"status"` // "ok" | "skipped" | "unknown"
	Valid       *bool  `json:"valid,omitempty"`
	LineType    string `json:"line_type,omitempty"`
	Carrier     string `json:"carrier,omitempty"`
	Country     string `json:"country,omitempty"` // numverify country_code (ISO alpha-2)
	CountryName string `json:"country_name,omitempty"`
	Location    string `json:"location,omitempty"`
	CheckedAt   string `json:"checked_at"`
	SourceTool  string `json:"source_tool"`
	Error       string `json:"error,omitempty"`
}

func (r NumverifyResult) lineTypeIfOK() string {
	if r.Status == StatusOK {
		return r.LineType
	}
	return ""
}

func (r NumverifyResult) carrierIfOK() string {
	if r.Status == StatusOK {
		return r.Carrier
	}
	return ""
}

func (r NumverifyResult) countryIfOK() string {
	if r.Status == StatusOK {
		return r.Country
	}
	return ""
}

// numverifyClient holds the resolved API key and endpoint. An empty apiKey means
// "no key configured" and yields a "skipped" result.
type numverifyClient struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// numverifyConfig is the optional JSON config file shape. Env vars take
// precedence over file values, so a config file can be committed as a template
// while real secrets stay in the environment.
type numverifyConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

// newNumverifyClientFromEnv builds a client from NUMVERIFY_API_KEY /
// NUMVERIFY_BASE_URL, optionally supplemented/overridden by NUMVERIFY_CONFIG.
// With no key set the numverify path is skipped cleanly and the module works
// with zero API keys configured.
func newNumverifyClientFromEnv() *numverifyClient {
	c := &numverifyClient{http: &http.Client{}}
	cfg := numverifyConfig{}

	if path := os.Getenv("NUMVERIFY_CONFIG"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "NUMVERIFY_CONFIG: failed to read %s: %v\n", path, err)
		} else {
			if err := json.Unmarshal(data, &cfg); err != nil {
				fmt.Fprintf(os.Stderr, "NUMVERIFY_CONFIG: failed to parse %s: %v\n", path, err)
			}
		}
	}

	if v := os.Getenv(APIKeyEnv); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv(BaseURLEnv); v != "" {
		cfg.BaseURL = v
	}

	c.apiKey = cfg.APIKey
	c.baseURL = cfg.BaseURL
	if c.baseURL == "" {
		c.baseURL = defaultNumverifyBaseURL
	}
	return c
}

// numverifyResponse is numverify's documented /validate response shape. It also
// carries an optional error object used by the API to report problems (e.g. an
// invalid access key) with an HTTP 200.
type numverifyResponse struct {
	Valid               bool   `json:"valid"`
	Number              string `json:"number"`
	InternationalFormat string `json:"international_format"`
	CountryCode         string `json:"country_code"`
	CountryName         string `json:"country_name"`
	Location            string `json:"location"`
	Carrier             string `json:"carrier"`
	LineType            string `json:"line_type"`
	Success             *bool  `json:"success"` // present (=false) only on error envelopes
	Error               *struct {
		Code int    `json:"code"`
		Type string `json:"type"`
		Info string `json:"info"`
	} `json:"error"`
}

// run performs the numverify lookup. With no API key it returns a clean
// "skipped" result. Any network/HTTP/API failure degrades to "unknown" with an
// error note — never a panic, never a blocked pipeline.
func (c *numverifyClient) run(ctx context.Context, phone string, timeout time.Duration, now time.Time) NumverifyResult {
	res := NumverifyResult{
		CheckedAt:  now.Format(time.RFC3339),
		SourceTool: NumverifyTool,
	}

	if c == nil || c.apiKey == "" {
		res.Status = StatusSkipped
		res.Error = fmt.Sprintf("%s not set and no %s — numverify carrier lookup skipped (local scanner still ran)", APIKeyEnv, "NUMVERIFY_CONFIG")
		return res
	}

	number := normalizePhone(phone)
	if number == "" {
		res.Status = StatusUnknown
		res.Error = "no phone field present on lead record"
		return res
	}

	q := url.Values{}
	q.Set("access_key", c.apiKey)
	q.Set("number", number)
	q.Set("format", "1")
	endpoint := c.baseURL + "?" + q.Encode()

	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(cctx, http.MethodGet, endpoint, nil)
	if err != nil {
		res.Status = StatusUnknown
		res.Error = "could not build numverify request: " + err.Error()
		return res
	}

	resp, err := c.http.Do(req)
	if err != nil {
		res.Status = StatusUnknown
		res.Error = "numverify request failed: " + err.Error()
		return res
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		res.Status = StatusUnknown
		res.Error = "reading numverify response: " + err.Error()
		return res
	}
	if resp.StatusCode != http.StatusOK {
		res.Status = StatusUnknown
		res.Error = fmt.Sprintf("numverify HTTP %d", resp.StatusCode)
		return res
	}

	var parsed numverifyResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		res.Status = StatusUnknown
		res.Error = "could not parse numverify JSON: " + err.Error()
		return res
	}
	// numverify signals problems in-band with success=false + an error object,
	// often on an HTTP 200 (e.g. invalid/exhausted key).
	if parsed.Error != nil || (parsed.Success != nil && !*parsed.Success) {
		msg := "numverify returned an error"
		if parsed.Error != nil {
			msg = fmt.Sprintf("numverify error %d (%s): %s", parsed.Error.Code, parsed.Error.Type, parsed.Error.Info)
		}
		res.Status = StatusUnknown
		res.Error = msg
		return res
	}

	valid := parsed.Valid
	res.Status = StatusOK
	res.Valid = &valid
	res.LineType = parsed.LineType
	res.Carrier = parsed.Carrier
	res.Country = parsed.CountryCode
	res.CountryName = parsed.CountryName
	res.Location = parsed.Location
	return res
}
