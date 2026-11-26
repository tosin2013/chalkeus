package evidence

import "testing"

func TestSCMAndPublisherID(t *testing.T) {
	r := &Run{State: map[string]any{}}
	if k, h := r.SCM(); k != "github" || h != "github.com" {
		t.Fatalf("pre-037 run: %s %s", k, h)
	}
	r.SetSCM("bitbucket", "bitbucket.org")
	if k, h := r.SCM(); k != "bitbucket" || h != "bitbucket.org" {
		t.Fatalf("%s %s", k, h)
	}
	r.State["publisher_login"] = "bot[bot]"
	if r.PublisherID() != "bot[bot]" {
		t.Fatal("falls back to the login")
	}
	r.SetPublisherID("{abc}")
	if r.PublisherID() != "{abc}" {
		t.Fatal("publisher id")
	}
}
