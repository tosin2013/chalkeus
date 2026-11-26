package scm

import "testing"

func TestValidRepo(t *testing.T) {
	ok := map[Kind][]string{
		GitHub:    {"owner/name", "o-1/n.x_y"},
		GitLab:    {"group/project", "a/b/c/project", "Group_1/sub.x/proj-2"},
		Bitbucket: {"workspace/repo", "ws_1/my-repo.x"},
	}
	bad := map[Kind][]string{
		GitHub:    {"a/b/c", "owner", "owner/"},
		GitLab:    {"project", "group/project.git", "group/-x", "group/.x", "g//p"},
		Bitbucket: {"Workspace/repo", "ws/a/b", "ws"},
	}
	for k, list := range ok {
		for _, r := range list {
			if err := ValidRepo(k, r); err != nil {
				t.Errorf("%s %q: %v", k, r, err)
			}
		}
	}
	for k, list := range bad {
		for _, r := range list {
			if err := ValidRepo(k, r); err == nil {
				t.Errorf("%s %q accepted", k, r)
			}
		}
	}
}

func TestHostNamesAndURLs(t *testing.T) {
	for _, s := range []string{"gitlab.example.com", "git.corp:8443", "localhost"} {
		if !ValidHostName(s) {
			t.Errorf("%q refused", s)
		}
	}
	for _, s := range []string{"https://gitlab.com", "gitlab.com/path", "", "-x.com", "a b"} {
		if ValidHostName(s) {
			t.Errorf("%q accepted", s)
		}
	}
	gl := NewHost(GitLab, "git.corp:8443", "")
	if gl.APIBase != "https://git.corp:8443/api/v4" || RepoURL(gl, "a/b/c") != "https://git.corp:8443/a/b/c.git" {
		t.Errorf("gitlab host %+v", gl)
	}
	if got := gl.EgressHosts(); len(got) != 1 || got[0] != "git.corp" {
		t.Errorf("egress %v", got)
	}
	if ChangeRequestURLPrefix(gl, "a/b") != "git.corp:8443/a/b/-/merge_requests/" {
		t.Error(ChangeRequestURLPrefix(gl, "a/b"))
	}
	bb := NewHost(Bitbucket, "", "")
	if bb.APIBase != "https://api.bitbucket.org/2.0" || ChangeRequestURLPrefix(bb, "ws/r") != "bitbucket.org/ws/r/pull-requests/" {
		t.Errorf("bitbucket %+v", bb)
	}
	gh := NewHost(GitHub, "", "")
	if ChangeRequestURLPrefix(gh, "o/n") != "github.com/o/n/pull/" || RepoURL(Host{}, "o/n") != "https://github.com/o/n.git" {
		t.Error("github urls")
	}
	if gh.OtherProbeHost() != "gitlab.com" || bb.OtherProbeHost() != "github.com" {
		t.Error("probe hosts")
	}
	if LastSegment("a/b/c-pilot-fixture") != "c-pilot-fixture" {
		t.Error("last segment")
	}
}

func TestParseKindAndAccounts(t *testing.T) {
	for in, want := range map[string]Kind{"": GitHub, "GitLab": GitLab, " bitbucket ": Bitbucket} {
		if k, err := ParseKind(in); err != nil || k != want {
			t.Errorf("%q: %v %v", in, k, err)
		}
	}
	if _, err := ParseKind("gitea"); err == nil {
		t.Error("gitea accepted")
	}
	if !(Account{ID: "Octo"}).Same(Account{ID: "octo"}) || (Account{}).Same(Account{}) {
		t.Error("account comparison")
	}
	r := TokenProbeResult{Lines: []ProbeLine{{OK: true}}}
	if !r.OK() {
		t.Error("ok probe")
	}
	r.Code = TokenWrongRepo
	if r.OK() {
		t.Error("coded probe passed")
	}
}
