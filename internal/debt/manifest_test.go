package debt

import (
	"reflect"
	"testing"
)

func TestLevel(t *testing.T) {
	for _, c := range []struct{ from, to, want string }{
		{"1.2.3", "1.2.4", LevelPatch},
		{"1.2.3", "1.3.0", LevelMinor},
		{"3.25.76", "4.1.12", LevelMajor},
		{"0.3.1", "0.4.0", LevelMajor},
		{"0.3.1", "0.3.2", LevelMinor},
		{"0.0.3", "0.0.4", LevelMajor},
		{"v1.9.0", "v1.10.0", LevelMinor},
		{"1.0.0-rc.1", "1.0.0", LevelPatch},
		{"1.2", "1.2.1", LevelPatch},
		{"latest", "1.0.0", LevelUnknown},
	} {
		if got := Level(c.from, c.to); got != c.want {
			t.Errorf("Level(%s, %s) = %s, want %s", c.from, c.to, got, c.want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
		ok   bool
	}{
		{"22.15.3", "22.10.1", 1, true},
		{"1.0.0", "1.0", 0, true},
		{"1.0.0-rc.1", "1.0.0", -1, true},
		{"v2.0.0", "2.0.0", 0, true},
		{"^1.0.0", "1.0.0", 0, false},
	} {
		got, ok := compareVersions(c.a, c.b)
		if got != c.want || ok != c.ok {
			t.Errorf("compareVersions(%s, %s) = %d, %v", c.a, c.b, got, ok)
		}
	}
}

const lockV3 = `{"lockfileVersion": 3, "packages": {
 "": {"name": "x", "dependencies": {"zod": "^3.25.0"}},
 "node_modules/zod": {"version": "3.25.76"},
 "node_modules/@types/node": {"version": "22.15.3"},
 "node_modules/a/node_modules/zod": {"version": "3.0.0"}}}`

func TestResolved(t *testing.T) {
	for _, c := range []struct{ file, content, pkg, want string }{
		{"package-lock.json", lockV3, "zod", "3.25.76"},
		{"package-lock.json", lockV3, "@types/node", "22.15.3"},
		{"package-lock.json", lockV3, "missing", ""},
		{"web/package-lock.json", `{"dependencies": {"left-pad": {"version": "1.3.0"}}}`, "left-pad", "1.3.0"},
		{"package.json", `{"dependencies": {"zod": "^3.25.0"}}`, "zod", ""},
		{"requirements.txt", "# pins\nRequests[socks]==2.32.3 ; python_version>'3'\nflask>=3\n", "requests", "2.32.3"},
		{"requirements-dev.txt", "Typing_Extensions==4.12.2\n", "typing-extensions", "4.12.2"},
		{"requirements.txt", "flask>=3\n", "flask", ""},
		{"go.mod", "module x\n\ngo 1.27\n\nrequire golang.org/x/net v0.30.0\n\nrequire (\n\tgithub.com/a/b v1.2.3 // indirect\n)\n", "github.com/a/b", "v1.2.3"},
		{"go.mod", "module x\nrequire golang.org/x/net v0.30.0\n", "golang.org/x/net", "v0.30.0"},
		{"pom.xml", `<project><properties><jackson.version>2.18.1</jackson.version></properties>
<dependencies><dependency><groupId>com.fasterxml.jackson.core</groupId><artifactId>jackson-databind</artifactId><version>${jackson.version}</version></dependency>
<dependency><groupId>junit</groupId><artifactId>junit</artifactId><version>[4.0,5.0)</version></dependency></dependencies></project>`,
			"com.fasterxml.jackson.core:jackson-databind", "2.18.1"},
		{"pom.xml", `<project><dependencies><dependency><groupId>junit</groupId><artifactId>junit</artifactId><version>[4.0,5.0)</version></dependency></dependencies></project>`, "junit:junit", ""},
		{"package-lock.json", "not json", "zod", ""},
	} {
		if got := Resolved(c.file, []byte(c.content), c.pkg); got != c.want {
			t.Errorf("Resolved(%s, %s) = %q, want %q", c.file, c.pkg, got, c.want)
		}
	}
}

func TestChanges(t *testing.T) {
	after := `{"lockfileVersion": 3, "packages": {"node_modules/zod": {"version": "4.1.12"},
 "node_modules/@types/node": {"version": "22.15.3"}, "node_modules/new": {"version": "1.0.0"}}}`
	got := Changes("package-lock.json", []byte(lockV3), []byte(after))
	want := []Package{{Name: "new", From: "", To: "1.0.0", Level: LevelUnknown},
		{Name: "zod", From: "3.25.76", To: "4.1.12", Level: LevelMajor}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v", got)
	}
}
