# Specification Quality Checklist: Working with an agent — pilot skill and harness approvals

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-11-14
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

- The spec names the two supported agent harnesses (Claude Code, OpenCode), the `pilot`
  skill, `AGENTS.md` and audit check ids (AU09, AU10). These are the user-facing product
  surface and audit contract being changed, as in specs 002–006, not internal design.
- Scope out: a `--yes` flag or any self-approval; approving merges outside GitHub; GitHub
  approvals (roadmap 004).
- No clarifications needed: the owner's decisions (2026-11-14) settle scope, the merge rule
  and the channel labelling.
