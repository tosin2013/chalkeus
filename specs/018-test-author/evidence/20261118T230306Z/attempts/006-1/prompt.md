You are a coding agent working in a git repository (the current directory).
Implement the specification below completely, then mark every task in the task list as
done by changing `- [ ]` to `- [X]` in that tasks.md file.

Rules:
- Change only files matching these paths (`*` also matches `/`):
  - `src/main/java/dev/stockroom/*`
  - `src/main/resources/*`
  - `src/test/java/dev/stockroom/*`
  - `specs/006-paging/tasks.md`
- The project uses the jdk21-maven toolchain. Check your work with: `mvn -B test`
- Add or extend tests for your change in the project's test files (`src/test/*`). Keep every existing
  test passing unchanged: the grader re-runs the original versions of any test file you edit or delete.
- Only the project's existing dependencies are available to the grader; do not add new ones.
- Do not run git commands; the harness commits your work.
- When finished, stop.

# Specification

# Feature Specification: Paging and filtering

**Feature Branch**: `006-paging`

**Created**: 2026-11-18

**Status**: Draft

**Input**: Roadmap item 006. List products a page at a time, optionally filtered by name, with
stable ordering. Depends on 004.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Page through the catalogue (Priority: P1)

As a client, I read the catalogue one page at a time, with totals so I know how many pages there
are.

**Why this priority**: A full list doesn't scale as the catalogue grows.

**Independent Test**: Create 5 products, read pages of size 2.

**Acceptance Scenarios**:

1. **Given** exactly 5 products, **When** a client calls `GET /api/products?page=0&size=2`,
   **Then** the response is `200` with `{"content": [<first two by id>], "page": 0, "size": 2,
   "totalElements": 5, "totalPages": 3}`.
2. **Given** the same products, **When** the client calls `page=2&size=2`, **Then** `content`
   holds only the fifth product; **when** it calls `page=3&size=2`, **Then** `content` is empty
   and the totals are unchanged.
3. **Given** only `size=2`, **When** it is called, **Then** `page` defaults to `0`; **given**
   only `page=0`, **Then** `size` defaults to `20`.

---

### User Story 2 - Filter by name (Priority: P1)

As a client, I find products whose name contains some text, ignoring case.

**Why this priority**: Finding a product by name is the most common lookup.

**Independent Test**: Create `Red Widget`, `Blue widget`, `Gadget`; filter by `widget`.

**Acceptance Scenarios**:

1. **Given** those three products, **When** a client calls `GET /api/products?name=widget`,
   **Then** the response is a page whose `content` holds `Red Widget` and `Blue widget` in id
   order, with `"totalElements": 2`.
2. **Given** `name=WIDGET&page=0&size=1`, **When** it is called, **Then** `content` holds only
   `Red Widget`, with `"totalElements": 2` and `"totalPages": 2`.
3. **Given** `name=` (empty or only spaces), **When** it is called, **Then** no filter applies.

---

### User Story 3 - The plain list stays as it is (Priority: P1)

As an existing client, my `GET /api/products` without any of these parameters still returns the
plain array.

**Why this priority**: Existing clients and tests must not break.

**Independent Test**: The 002–004 list tests pass unedited.

**Acceptance Scenarios**:

1. **Given** products, **When** a client calls `GET /api/products` with none of `name`, `page`
   or `size`, **Then** the response is the JSON array of all products in id order, as in 002.

---

### User Story 4 - Bad paging parameters (Priority: P2)

As a client, I get a Problem Detail when my paging parameters make no sense.

**Why this priority**: Unbounded or negative pages would be errors or a performance risk.

**Independent Test**: Call with out-of-range and non-numeric values.

**Acceptance Scenarios**:

1. **Given** `page=-1`, `size=0`, `size=101` or `size=abc`, **When** it is called, **Then** the
   response is `400` with `Content-Type: application/problem+json`, `"status": 400` and
   `"instance": "/api/products"` (as in 003).

---

### Edge Cases

- Ordering is ascending id, so pages are stable while no product is added.
- `name` is trimmed before matching; it matches anywhere in the name, ignoring case.
- A page past the end is `200` with empty `content`, not an error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `GET /api/products` with any of `name`, `page` or `size` MUST answer `200` with
  `{"content": [products], "page": int, "size": int, "totalElements": long, "totalPages": int}`,
  products in ascending id order.
- **FR-002**: `page` MUST default to `0` and be `>= 0`; `size` MUST default to `20` and be
  between `1` and `100`. An out-of-range or non-numeric value MUST answer `400` as a Problem
  Detail (003 FR-001).
- **FR-003**: `name`, when not blank after trimming, MUST keep only products whose name contains
  it, ignoring case; `totalElements` and `totalPages` count only those.
- **FR-004**: `GET /api/products` with none of the three parameters MUST answer exactly as in
  002: the plain array of all products in id order.
- **FR-005**: The filtering and paging MUST happen in the database query, not by loading every
  product into memory.
- **FR-006**: Tests for this item MUST be added under `src/test/java/dev/stockroom/`, and every
  existing test MUST keep passing unedited. No dependency may be added.

## Allowed paths

- `src/main/java/dev/stockroom/*`
- `src/main/resources/*`
- `src/test/java/dev/stockroom/*`

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every acceptance scenario above passes as an automated test under `mvn test`.
- **SC-002**: The 001–005 tests pass without being edited (or 001–004, if 006 runs before 005).

## Assumptions

- Scenarios that count products assume the test controls the table's contents (it clears
  `product` first); tests share one in-memory database per Spring context.
- The page shape is Stockroom's own record, not Spring Data's `Page` serialisation, so it stays
  stable across framework versions.

# Plan

# Implementation Plan: Paging and filtering

**Branch**: `006-paging` | **Date**: 2026-11-18 | **Spec**: [spec.md](spec.md)

## Summary

A second handler on `GET /api/products`, selected when any of `name`, `page`, `size` is present
(`@GetMapping(params = …)`, or one handler that branches on them). Validated `@RequestParam`s
(`@Min(0)`, `@Min(1) @Max(100)`) give `400` through the 003 handler. `ProductRepository` adds
`findByNameContainingIgnoreCase(String, Pageable)` and `countByNameContainingIgnoreCase`
(or `ListPagingAndSortingRepository` for the unfiltered case), sorted by id. The response is a
`ProductPage` record with the five fields.

## Technical Context

**Language/Version**: Java 21 | **Framework**: Spring Boot 4.1.2 (`webmvc`, `validation`,
`data-jdbc`, embedded H2) | **Testing**: JUnit 5, `@SpringBootTest` + MockMvc, `JdbcTemplate` to
clear the table | **Build**: `mvn test`
**Constraints**: no new dependencies; tests offline; existing tests unchanged

## Constitution Check

I item 006, depends on 004 ✓ · II owner merges ✓ · III one PR with tests ✓ · IV no dependency
change ✓ · V offline tests ✓ · VI no planning files or CI touched ✓

## Project Structure

```text
src/main/java/dev/stockroom/product/ProductPage.java
src/main/java/dev/stockroom/product/{ProductRepository,ProductStore,ProductController}.java
src/test/java/dev/stockroom/product/ProductPagingTests.java
```

# Task list (`specs/006-paging/tasks.md`)

# Tasks: Paging and filtering

**Input**: `specs/006-paging/` (spec.md, plan.md)

- [ ] T001 [US1] Add record `ProductPage(List<Product> content, int page, int size, long totalElements, int totalPages)` in `src/main/java/dev/stockroom/product/` (FR-001)
- [ ] T002 [US2] Add paged, case-insensitive name queries ordered by id to `ProductRepository`, and a `findPage(String name, int page, int size)` to `ProductStore`, in `src/main/java/dev/stockroom/product/` (FR-001, FR-003, FR-005)
- [ ] T003 [US1] In `src/main/java/dev/stockroom/product/ProductController.java`, answer `GET /api/products` with a `ProductPage` when `name`, `page` or `size` is present, and the plain array otherwise (FR-001, FR-004)
- [ ] T004 [US4] Validate `page` (`>= 0`, default 0) and `size` (1–100, default 20) so bad values answer `400` Problem Details through `src/main/java/dev/stockroom/ApiExceptionHandler.java` (FR-002)
- [ ] T005 Add tests in `src/test/java/dev/stockroom/product/ProductPagingTests.java` for every acceptance scenario of US1–US4 (FR-006)
- [ ] T006 Run `mvn test` and confirm the earlier items' tests pass unedited (SC-002)
