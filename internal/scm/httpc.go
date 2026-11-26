package scm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/tosin2013/chalkeus/internal/secret"
	"github.com/tosin2013/chalkeus/internal/verbose"
)

// PageCap is the most items a list read follows pagination to; a capped list says so.
const PageCap = 500

// HTTP is the small REST client the GitLab and Bitbucket implementations share (research R2):
// base URL, auth, proxy from the environment, an optional CA bundle, a timeout, pagination,
// status mapping and token redaction.
type HTTP struct {
	Base   string // API base, without a trailing slash
	Token  secret.Value
	Basic  string // Basic-auth username; "" sends Authorization: Bearer
	Client *http.Client
}

// NewHTTP builds an HTTP client for h with token. A CA bundle on h is trusted in addition to the
// system roots.
func NewHTTP(h Host, token secret.Value) (*HTTP, error) {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = http.ProxyFromEnvironment
	if h.CABundle != "" {
		pool, err := LoadCABundle(h.CABundle)
		if err != nil {
			return nil, err
		}
		tr.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	}
	host := hostOf(h.APIBase)
	c := &http.Client{Transport: tr, Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			if req.URL.Host != host {
				return fmt.Errorf("redirect to another host (%s) refused", req.URL.Host)
			}
			return nil
		}}
	if !token.Empty() {
		verbose.AddRedaction(token.Reveal())
	}
	return &HTTP{Base: strings.TrimRight(h.APIBase, "/"), Token: token, Client: c}, nil
}

// LoadCABundle reads a PEM bundle into a pool that also holds the system roots.
func LoadCABundle(path string) (*x509.CertPool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("FORGE_SCM_CA_BUNDLE: %v", err)
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(raw) {
		return nil, fmt.Errorf("FORGE_SCM_CA_BUNDLE: %s holds no PEM certificate", path)
	}
	return pool, nil
}

func hostOf(base string) string {
	u, err := url.Parse(base)
	if err != nil {
		return ""
	}
	return u.Host
}

// Do sends one request. path is relative to Base (or absolute when it starts with http). body,
// when not nil, is sent as JSON unless it is an io.Reader (then contentType is used). out, when
// not nil, receives the decoded JSON. It returns the response headers.
func (c *HTTP) Do(ctx context.Context, method, path string, body any, contentType string, out any) (http.Header, error) {
	u := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		u = c.Base + "/" + strings.TrimLeft(path, "/")
	}
	var rd io.Reader
	switch b := body.(type) {
	case nil:
	case io.Reader:
		rd = b
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			return nil, err
		}
		rd = strings.NewReader(string(raw))
		contentType = "application/json"
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return nil, &Error{Code: ErrInvalid, Msg: c.redact(err.Error())}
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	if !c.Token.Empty() {
		if c.Basic != "" {
			req.SetBasicAuth(c.Basic, c.Token.Reveal())
		} else {
			req.Header.Set("Authorization", "Bearer "+c.Token.Reveal())
		}
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, &Error{Code: ErrUnavailable, Msg: c.redact(fmt.Sprintf("%s %s: %v", method, c.shortURL(u), err))}
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode >= 300 {
		return resp.Header, c.statusErr(method, u, resp, raw)
	}
	if out != nil && len(raw) > 0 {
		if b, ok := out.(*[]byte); ok {
			*b = raw
			return resp.Header, nil
		}
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.Header, &Error{Code: ErrInvalid, Msg: fmt.Sprintf("%s %s: unexpected response: %v", method, c.shortURL(u), err)}
		}
	}
	return resp.Header, nil
}

var linkNextRE = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

// List follows pagination: a Link rel="next" header (GitHub, GitLab) or a JSON "next" field
// (Bitbucket, via next). page decodes one page and returns its item count and the next URL found
// in the body ("" for none). It stops at PageCap items and reports capped.
func (c *HTTP) List(ctx context.Context, path string, page func(raw []byte) (int, string, error)) (capped bool, err error) {
	next, total := path, 0
	for next != "" {
		var raw []byte
		h, err := c.Do(ctx, http.MethodGet, next, nil, "", &raw)
		if err != nil {
			return false, err
		}
		n, bodyNext, err := page(raw)
		if err != nil {
			return false, err
		}
		total += n
		if total >= PageCap {
			return true, nil
		}
		next = bodyNext
		if next == "" {
			if m := linkNextRE.FindStringSubmatch(h.Get("Link")); m != nil {
				next = m[1]
			}
		}
		if n == 0 {
			break
		}
	}
	return false, nil
}

func (c *HTTP) statusErr(method, u string, resp *http.Response, raw []byte) error {
	code := ErrUnavailable
	switch s := resp.StatusCode; {
	case s == 401:
		code = ErrUnauthorized
	case s == 403:
		code = ErrForbidden
		if resp.Header.Get("RateLimit-Remaining") == "0" || resp.Header.Get("X-RateLimit-Remaining") == "0" {
			code = ErrRateLimited
		}
	case s == 404:
		code = ErrNotFound
	case s == 409:
		code = ErrConflict
	case s == 429:
		code = ErrRateLimited
	case s == 400 || s == 422:
		code = ErrInvalid
	}
	msg := strings.TrimSpace(string(raw))
	if len(msg) > 300 {
		msg = msg[:300] + "…"
	}
	return &Error{Code: code, Status: resp.StatusCode,
		Msg: c.redact(fmt.Sprintf("%s %s: HTTP %d %s", method, c.shortURL(u), resp.StatusCode, msg))}
}

// shortURL drops the query (it can carry user ids) and the base, for messages.
func (c *HTTP) shortURL(u string) string {
	if i := strings.Index(u, "?"); i >= 0 {
		u = u[:i]
	}
	return strings.TrimPrefix(u, c.Base)
}

func (c *HTTP) redact(s string) string {
	if !c.Token.Empty() {
		s = strings.ReplaceAll(s, c.Token.Reveal(), "[redacted]")
	}
	return verbose.Redact(s)
}

// PathEscape escapes a value for one path segment, slashes included (GitLab project paths).
func PathEscape(s string) string { return url.PathEscape(s) }
