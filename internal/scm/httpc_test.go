package scm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/secret"
)

const tok = "glpat-SECRETSECRETSECRET"

func testHTTP(t *testing.T, h http.HandlerFunc) *HTTP {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := NewHTTP(Host{Kind: GitLab, Name: "x", APIBase: srv.URL + "/api/v4"}, secret.New(tok))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestHTTPAuthAndDecode(t *testing.T) {
	c := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+tok || r.URL.Path != "/api/v4/projects/a%2Fb" && r.URL.RawPath != "/api/v4/projects/a%2Fb" {
			http.Error(w, "bad", 400)
			return
		}
		_, _ = fmt.Fprint(w, `{"default_branch":"main"}`)
	})
	var out struct {
		DefaultBranch string `json:"default_branch"`
	}
	if _, err := c.Do(context.Background(), "GET", "projects/"+PathEscape("a/b"), nil, "", &out); err != nil || out.DefaultBranch != "main" {
		t.Fatalf("%v %+v", err, out)
	}
}

func TestHTTPErrorsNeverCarryTheToken(t *testing.T) {
	c := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = fmt.Fprintf(w, `{"message":"token %s is not allowed"}`, tok)
	})
	_, err := c.Do(context.Background(), "GET", "user", nil, "", nil)
	if !IsCode(err, ErrForbidden) || strings.Contains(err.Error(), tok) || !strings.Contains(err.Error(), "[redacted]") {
		t.Fatalf("%v", err)
	}
}

func TestHTTPStatusMapping(t *testing.T) {
	for status, code := range map[int]string{401: ErrUnauthorized, 404: ErrNotFound, 409: ErrConflict, 422: ErrInvalid, 429: ErrRateLimited, 500: ErrUnavailable} {
		c := testHTTP(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) })
		if _, err := c.Do(context.Background(), "GET", "x", nil, "", nil); !IsCode(err, code) {
			t.Errorf("%d: %v", status, err)
		}
	}
	c := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("RateLimit-Remaining", "0")
		w.WriteHeader(403)
	})
	if _, err := c.Do(context.Background(), "GET", "x", nil, "", nil); !IsCode(err, ErrRateLimited) {
		t.Errorf("403 with no remaining: %v", err)
	}
}

func TestHTTPListFollowsLinkAndBodyNext(t *testing.T) {
	var srvURL string
	c := testHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v4/items?page=2>; rel="next"`, srvURL))
			_, _ = fmt.Fprint(w, `{"values":[1,2],"next":""}`)
		case "2":
			_, _ = fmt.Fprintf(w, `{"values":[3],"next":"%s/api/v4/items?page=3"}`, srvURL)
		default:
			_, _ = fmt.Fprint(w, `{"values":[4]}`)
		}
	})
	srvURL = strings.TrimSuffix(c.Base, "/api/v4")
	var got []int
	capped, err := c.List(context.Background(), "items", func(raw []byte) (int, string, error) {
		var p struct {
			Values []int  `json:"values"`
			Next   string `json:"next"`
		}
		if err := jsonUnmarshal(raw, &p); err != nil {
			return 0, "", err
		}
		got = append(got, p.Values...)
		return len(p.Values), p.Next, nil
	})
	if err != nil || capped || len(got) != 4 {
		t.Fatalf("%v %v %v", got, capped, err)
	}
}

func TestHTTPRefusesCrossHostRedirect(t *testing.T) {
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = fmt.Fprint(w, "{}") }))
	defer other.Close()
	c := testHTTP(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, other.URL+"/x", http.StatusFound) })
	if _, err := c.Do(context.Background(), "GET", "x", nil, "", nil); err == nil || !strings.Contains(err.Error(), "another host") {
		t.Fatalf("%v", err)
	}
}

func TestLoadCABundleRefusesNonPEM(t *testing.T) {
	p := t.TempDir() + "/ca.pem"
	if err := writeFile(p, "not a cert"); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCABundle(p); err == nil || !strings.Contains(err.Error(), "FORGE_SCM_CA_BUNDLE") {
		t.Fatalf("%v", err)
	}
}
