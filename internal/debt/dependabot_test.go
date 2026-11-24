package debt

import (
	"reflect"
	"testing"
)

func names(b Bump) []string {
	var out []string
	for _, p := range b.Packages {
		out = append(out, p.Name+" "+p.From+">"+p.To+" "+p.Level)
	}
	return out
}

func TestParseBump(t *testing.T) {
	for _, c := range []struct {
		name string
		pr   PR
		eco  string
		tc   string
		dir  string
		want []string
	}{
		{"single with prefix", PR{Title: "chore(deps): bump fast-uri from 3.1.0 to 3.1.5", HeadRef: "dependabot/npm_and_yarn/fast-uri-3.1.5"},
			"npm_and_yarn", "node24", "/", []string{"fast-uri 3.1.0>3.1.5 patch"}},
		{"single in a directory", PR{Title: "chore(deps): bump postcss from 8.5.13 to 8.5.25 in /docs", HeadRef: "dependabot/npm_and_yarn/docs/postcss-8.5.25"},
			"npm_and_yarn", "node24", "/docs", []string{"postcss 8.5.13>8.5.25 patch"}},
		{"major", PR{Title: "Bump zod from 3.25.76 to 4.1.12", HeadRef: "dependabot/npm_and_yarn/zod-4.1.12"},
			"npm_and_yarn", "node24", "/", []string{"zod 3.25.76>4.1.12 major"}},
		{"two packages from the body", PR{Title: "chore(deps): bump esbuild and tsx", HeadRef: "dependabot/npm_and_yarn/multi-abc",
			Body: "Bumps [esbuild](https://x) to 0.28.1 and updates ancestor dependency [tsx](https://y).\n\nUpdates `esbuild` from 0.25.9 to 0.28.1\n<details>\n\nUpdates `tsx` from 4.20.5 to 4.21.0\n<details>"},
			"npm_and_yarn", "node24", "/", []string{"esbuild 0.25.9>0.28.1 major", "tsx 4.20.5>4.21.0 minor"}},
		{"no versions in the title", PR{Title: "chore(deps): bump ws in /docs", HeadRef: "dependabot/npm_and_yarn/docs/multi-x",
			Body: "Bumps the npm_and_yarn group with 1 update in the /docs directory: [ws](https://z).\n\nUpdates `ws` from 7.5.10 to 8.19.0"},
			"npm_and_yarn", "node24", "/docs", []string{"ws 7.5.10>8.19.0 major"}},
		{"grouped table", PR{Title: "Bump the prod group across 1 directory with 2 updates", HeadRef: "dependabot/pip/prod-abc",
			Body: "| Package | From | To |\n| --- | --- | --- |\n| [requests](https://a) | `2.32.0` | `2.32.3` |\n| [flask](https://b) | `3.0.0` | `3.1.0` |\n"},
			"pip", "python3.13", "/", []string{"requests 2.32.0>2.32.3 patch", "flask 3.0.0>3.1.0 minor"}},
		{"bumps sentence", PR{Title: "Bump golang.org/x/net", HeadRef: "dependabot/go_modules/golang.org/x/net-0.31.0",
			Body: "Bumps [golang.org/x/net](https://c) from 0.30.0 to 0.31.0.\n"},
			"go_modules", "go", "/", []string{"golang.org/x/net 0.30.0>0.31.0 major"}},
		{"ecosystem without a toolchain", PR{Title: "Bump actions/checkout from 4 to 5", HeadRef: "dependabot/github_actions/actions/checkout-5"},
			"github_actions", "", "/", []string{"actions/checkout 4>5 major"}},
	} {
		b, ok := ParseBump(c.pr)
		if !ok || b.Ecosystem != c.eco || b.Toolchain != c.tc || b.Directory != c.dir || !reflect.DeepEqual(names(b), c.want) {
			t.Errorf("%s: ok=%v %+v %v", c.name, ok, b, names(b))
		}
	}
	if _, ok := ParseBump(PR{Title: "Update the README", HeadRef: "dependabot/npm_and_yarn/x"}); ok {
		t.Error("a title without a bump parsed")
	}
}
