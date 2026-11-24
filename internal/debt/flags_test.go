package debt

import (
	"strings"
	"testing"
)

func TestFlags(t *testing.T) {
	m := &memReader{checks: map[string][]Check{"MAIN": {{Name: "ci", State: CheckFailing}}},
		files: map[string]string{
			"package.json":      "{\n \"scripts\": {\n  \"test\": \"mocha --recursive --exit\",\n  \"lint\": \"eslint .\"\n }\n}\n",
			"jest.config.ts":    "export default {\n  testEnvironment: 'node',\n  forceExit: true,\n};\n",
			"package-lock.json": `{"packages":{"node_modules/sqlite3":{"hasInstallScript":true},"node_modules/a/node_modules/bcrypt":{"hasInstallScript":true},"node_modules/x":{}}}`,
		}}
	rep, _ := build(t, m)
	got := map[string]string{}
	for _, f := range rep.BaselineFlags {
		got[f.Kind+" "+f.Where] = f.Detail
	}
	for _, want := range []string{"red default branch ci", "force exit package.json:3", "force exit jest.config.ts:3", "install scripts package-lock.json"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing %q in %v", want, got)
		}
	}
	if d := got["install scripts package-lock.json"]; !strings.Contains(d, "2 packages") || !strings.Contains(d, "bcrypt, sqlite3") {
		t.Error(d)
	}
}

func TestNoFlags(t *testing.T) {
	m := &memReader{checks: map[string][]Check{"MAIN": pass("ci")}, files: map[string]string{"package.json": `{"scripts":{"test":"jest"}}`}}
	rep, _ := build(t, m)
	if len(rep.BaselineFlags) != 0 {
		t.Fatal(rep.BaselineFlags)
	}
}
