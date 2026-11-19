# Implementation Plan: Greenfield bootstrap

**Branch**: `017-greenfield` | **Date**: 2026-11-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/017-greenfield/spec.md`. Written after the work,
as its record, like [tasks.md](tasks.md). 017 was built guide-first on a real repository, so
this plan records the approach that was taken and leaves no work open.

## Summary

A new project is a 014 target whose first item is a scaffold. 017 adds no new run kind. It adds:
- the order of the work: everything is planned before anything is built;
- a Maven prefetch that works before the code exists;
- a guide.

[Stockroom](https://github.com/tosin2013/stockroom) is the reference: a Spring Boot inventory
service, built from an empty repository to 8 of 8 items verified.

## Technical Context

- **Pilot:** Go (`cmd/pilot`, `internal/`), unchanged in its run kinds; 017 uses 014's item
  runs as they are.
- **Toolchain:** `jdk21-maven` (`pilot/images/toolchain-jdk21-maven/`), with Maven Central as the
  one registry.
- **Reference project:** Java 21, Spring Boot 4.1, start.spring.io starters, offline tests under
  `mvn -B test`.
- **Constraints:** every dependency is declared in the planning PR (the grader is offline, 014
  R7). The planning files, `pom.xml` and CI stay outside every item's allowed paths.

## Constitution Check

- **C-12 (humans plan, agents build):** the owner planned Stockroom with the controller.
  The constitution, roadmap, first specs and build file went in one planning PR the owner
  merged ✓
- **C-04 (human merge only):** every item arrived as a draft PR the owner merged ✓
- **C-07, C-08 (isolation):** unchanged from 014 ✓
- **C-14 (hidden tests):** drafted by the test author in a separate session, merged by the
  owner ✓

## Approach

1. **Planning PR** (FR-002, FR-005): stockroom#1. It holds the constitution, the roadmap, specs
   001 and 002, a `pom.xml` with every planned dependency, and CI.
2. **Prefetch before the code exists** (FR-003): `pilot-maven-prefetch` runs the tests with the
   hidden ones in place and ignores failures, so Maven downloads its test plugins. When the
   hidden tests don't compile yet, it reruns the test phase with one trivial test in their
   place.
3. **Scaffold, then dependent items** (FR-004, FR-007): items run as in 014. A change to what an
   earlier test expects is declared in the item's spec (`## Tests that may change`, check G7).
4. **Guide** (FR-006): [docs/greenfield.md](../../docs/greenfield.md), linked from
   [docs/targets.md](../../docs/targets.md).
5. **Cluster profiles** (FR-009): validated on kind. The OpenShift part moved to roadmap item 023
   (owner decision 2026-11-19).

## Project Structure

```text
pilot/images/toolchain-jdk21-maven/   pilot-maven-prefetch
pilot/images/toolchain-samples/       the offline Maven sample it is tested with
docs/greenfield.md                    the guide
specs/017-greenfield/                 spec, plan, tasks, debrief
```
