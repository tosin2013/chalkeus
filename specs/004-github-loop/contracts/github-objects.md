# Contract: GitHub objects the pilot writes and reads

All writes use the publisher credential, in a child process or the publisher Job. All reads use
the operator's `gh` in laptop mode.

## Labels

`status:undecided`, `status:needs-info`, `status:planned`, `status:specced`,
`status:in-progress`, `status:implemented`, `status:verified`, `status:deferred`,
`status:abandoned`. An item issue carries exactly one. Other labels are left alone.

## Item issue body

```markdown
<!-- specfoundry:item=001 -->
**001 — Slug function** · status `planned` · spec `specs/001-slugify`

<description>

**Outcome:** <outcome>

### Tasks (from specs/001-slugify/tasks.md)
- [x] T001 …
- [ ] T002 …

_Generated from the roadmap by specfoundry. Edits here are overwritten by `pilot github sync`._
```

## Decision post (tracking issue comment)

```markdown
<!-- specfoundry:decision run=20261116T101500Z step=retry task=A n=2 -->
**Decision pending: retry task A (attempt 2).** Attempt 1 failed: <reason>.

To approve, reply with exactly:

    /approve retry

Allowed approvers: @alice. To cancel your approval before the run resumes: `/withdraw retry`.
```

## Approval comment

- The first line must be exactly `/approve mode` or `/approve retry`, case-sensitive, with no
  extra words.
- It must be posted after the matching decision post, and must never have been edited.
- `/withdraw <step>` by the same author, posted before the run resumes, cancels it.

## Status-change PR

- **Title:** `pilot <run-id>: <item> verified`
- **Draft:** yes
- **Head:** `pilot/<run-id>/status-<item>`
- **Change:** one commit changing exactly the item's heading in `.specify/memory/roadmap.md`
- **Body:** names the item issue, and ends with: "Merging this PR approves the status change.
  The pilot checks that the merged change is exactly this edit."
