#!/bin/bash
# T053 step 1: one readiness refusal per rule, on local copies of Stockroom at 79d337e (008 specced).
# Nothing is pushed; each copy gets one local commit that breaks one rule.
set -u
SR=$1; R=$2; cd "$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
mk() { rm -rf "$R/$1"; git clone -q "$SR" "$R/$1"; git -C "$R/$1" checkout -q 79d337e; git -C "$R/$1" -c user.name=t053 -c user.email=t053@invalid checkout -q -b t053; }
commit() { git -C "$R/$1" add -A; git -C "$R/$1" -c user.name=t053 -c user.email=t053@invalid commit -q -m "t053: $2"; }
check() { # name item expected
  out=$(pilot/pilot gate --target stockroom --item "$2" --root "$R/$1" 2>&1); code=$?
  got=$(printf '%s' "$out" | python3 -c '
import json,sys
t=sys.stdin.read(); i=t.find("{")
try:
  d=json.loads(t[i:]); print(d.get("decision"), [c["detail"].split(":")[0] if c["id"]=="gate" else c["id"] for c in d.get("checks",[]) if c.get("result")=="fail"][:3], d.get("reasons", "")[:1] if isinstance(d.get("reasons"),list) else "")
except Exception as e: print("unparsed:", t[-200:].replace("\n"," "))')
  printf '%-22s item=%s exit=%s expect=%-24s got=%s\n' "$1" "$2" "$code" "$3" "$got"
}
mk control;                                                                          check control 008 allow
mk not-on-roadmap;                                                                   check not-on-roadmap 099 ITEM_NOT_ON_ROADMAP
mk spec-missing;  git -C $R/spec-missing rm -q specs/008-filter-wildcards/spec.md;  commit spec-missing "rm spec";   check spec-missing 008 ITEM_SPEC_MISSING
mk tasks-missing; git -C $R/tasks-missing rm -q specs/008-filter-wildcards/tasks.md; commit tasks-missing "rm tasks"; check tasks-missing 008 ITEM_TASKS_MISSING
mk open-q; printf '\n- [NEEDS CLARIFICATION: is a backslash literal?]\n' >> $R/open-q/specs/008-filter-wildcards/spec.md; commit open-q "open question"; check open-q 008 ITEM_OPEN_QUESTIONS
mk no-paths; python3 - "$R/no-paths/specs/008-filter-wildcards/spec.md" <<'P'
import sys,re;p=sys.argv[1];t=open(p).read();t=re.sub(r"## Allowed paths\n.*?(?=\n## )","",t,flags=re.S);open(p,"w").write(t)
P
commit no-paths "no allowed paths"; check no-paths 008 ITEM_NO_ALLOWED_PATHS
mk wide-paths; python3 - "$R/wide-paths/specs/008-filter-wildcards/spec.md" <<'P'
import sys;p=sys.argv[1];t=open(p).read();t=t.replace("- `src/main/java/dev/stockroom/*`","- `*`",1);open(p,"w").write(t)
P
commit wide-paths "allowed path *"; check wide-paths 008 ITEM_PATHS_TOO_WIDE
mk no-hidden; mv $R/no-hidden/specs/008-filter-wildcards $R/no-hidden/specs/009-filter-wildcards; sed -i 's/### 008 — The name filter/### 009 — The name filter/; s#specs/008-filter-wildcards/#specs/009-filter-wildcards/#' $R/no-hidden/.specify/memory/roadmap.md; commit no-hidden "item 009 without hidden tests"; check no-hidden 009 ITEM_NO_HIDDEN_TESTS
mk too-large; head -c 60M /dev/urandom > $R/too-large/blob.bin; commit too-large "60 MB file"; check too-large 008 INPUT_TOO_LARGE
mk dep; sed -i 's/### 006 — Paging and filtering  \[status: verified\]/### 006 — Paging and filtering  [status: specced]/' $R/dep/.specify/memory/roadmap.md; commit dep "006 not verified"; check dep 008 DEP_UNSATISFIED:006
mk unreadable; printf 'garbage\n' > $R/unreadable/.specify/memory/roadmap.md; commit unreadable "roadmap unreadable"; check unreadable 008 "STATUS/SCHEMA (unreadable)"
mk no-spec-dir; sed -i '/^- \*\*Spec dir:\*\* specs\/008-filter-wildcards\/$/d' $R/no-spec-dir/.specify/memory/roadmap.md; commit no-spec-dir "no spec dir"; check no-spec-dir 008 ITEM_SPEC_MISSING
