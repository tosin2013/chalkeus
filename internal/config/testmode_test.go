package config

import "testing"

func TestTestMode(t *testing.T) {
	t.Setenv("PILOT_TEST_MODE", "")
	t.Setenv("PILOT_TEST_GITHUB_API", "")
	t.Setenv("PILOT_TEST_GIT_BASE", "")
	if tm, err := TestMode(); err != nil || tm.On || tm.Record() != nil {
		t.Fatalf("off by default: %+v %v", tm, err)
	}
	t.Setenv("PILOT_TEST_GITHUB_API", "http://172.18.0.1:8090")
	if _, err := TestMode(); err == nil || err.(*Error).ExitCode() != 2 {
		t.Fatalf("an address without test mode is refused: %v", err)
	}
	t.Setenv("PILOT_TEST_MODE", "1")
	t.Setenv("PILOT_TEST_GIT_BASE", "http://172.18.0.1:8090/git")
	tm, err := TestMode()
	if err != nil || !tm.On || tm.GitBase != "http://172.18.0.1:8090/git/" {
		t.Fatalf("%+v %v", tm, err)
	}
	if r := tm.Record(); r["test_mode"] != true || r["test_github_api"] != "http://172.18.0.1:8090" {
		t.Fatalf("%v", r)
	}
	t.Setenv("PILOT_TEST_GITHUB_API", "file:///etc")
	if _, err := TestMode(); err == nil {
		t.Fatal("only http(s)")
	}
}
