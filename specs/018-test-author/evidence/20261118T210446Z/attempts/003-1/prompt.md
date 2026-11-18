You are a coding agent working in a git repository (the current directory).
Implement the specification below completely, then mark every task in the task list as
done by changing `- [ ]` to `- [X]` in that tasks.md file.

Rules:
- Change only files matching these paths (`*` also matches `/`):
  - `src/main/java/dev/stockroom/*`
  - `src/main/resources/*`
  - `src/test/java/dev/stockroom/*`
  - `specs/003-problem-details/tasks.md`
- The project uses the jdk21-maven toolchain. Check your work with: `mvn -B test`
- Add or extend tests for your change in the project's test files (`src/test/*`). Keep every existing
  test passing unchanged: the grader re-runs the original versions of any test file you edit or delete.
- Only the project's existing dependencies are available to the grader; do not add new ones.
- Do not run git commands; the harness commits your work.
- When finished, stop.

# Specification

# Feature Specification: Problem Details errors

**Feature Branch**: `003-problem-details`

**Created**: 2026-11-18

**Status**: Draft

**Input**: Roadmap item 003. Errors from the products API become RFC 9457 Problem Details:
validation failures answer `400` with one entry per invalid field, and an unknown product
answers `404`. Depends on 002.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See which fields are wrong (Priority: P1)

As a client, when I send a product the catalogue can't accept, I get a `400` Problem Detail that
names every invalid field, so I can fix them all at once.

**Why this priority**: A bare `400` says something is wrong but not what.

**Independent Test**: Post invalid bodies to `/api/products` and read the `errors` list.

**Acceptance Scenarios**:

1. **Given** the body `{"name": "", "stock": -1}`, **When** it is posted to `/api/products`,
   **Then** the response is `400` with `Content-Type: application/problem+json` and a body with
   `"status": 400`, `"title": "Bad Request"`, a non-empty `detail`, `"instance":
   "/api/products"`, and `errors` holding exactly two entries, `name` then `stock`, each with a
   non-empty `message`.
2. **Given** the body `{"name": "Widget"}` (no `stock`), **When** it is posted, **Then**
   `errors` holds exactly one entry, for `stock`.
3. **Given** a `name` of 101 characters and `"stock": 5`, **When** it is posted, **Then**
   `errors` holds exactly one entry, for `name`.
4. **Given** any of these, **When** the list is read afterwards, **Then** nothing was stored.

---

### User Story 2 - Unknown product (Priority: P1)

As a client, when I ask for a product that doesn't exist, I get a `404` Problem Detail that says
which id was not found.

**Why this priority**: Clients need to tell "not found" apart from other failures, in the same
format as every other error.

**Independent Test**: `GET /api/products/999` on an empty catalogue.

**Acceptance Scenarios**:

1. **Given** no product with id `999`, **When** a client calls `GET /api/products/999`, **Then**
   the response is `404` with `Content-Type: application/problem+json` and a body with
   `"status": 404`, `"title": "Not Found"`, a `detail` that contains `999`, and `"instance":
   "/api/products/999"`.

---

### User Story 3 - Unreadable requests (Priority: P2)

As a client, when my request can't even be read, I still get a Problem Detail rather than an
empty body.

**Why this priority**: Every error the API returns should have one shape.

**Independent Test**: Send broken JSON and a non-numeric id.

**Acceptance Scenarios**:

1. **Given** the body `{"name": ` (malformed JSON), **When** it is posted to `/api/products`,
   **Then** the response is `400` with `Content-Type: application/problem+json`, `"status":
   400` and `"instance": "/api/products"`.
2. **Given** the path `/api/products/abc`, **When** a client calls `GET` on it, **Then** the
   response is `400` with `Content-Type: application/problem+json`, `"status": 400` and
   `"instance": "/api/products/abc"`.

---

### Edge Cases

- A field that breaks more than one rule still gets exactly one entry in `errors`.
- Successful responses don't change: same status codes, bodies and `Content-Type:
  application/json` as in 002.
- Paths outside `/api/products` (for example the actuator endpoints 001 hides) keep their
  current status codes; their body format is not specified here.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every error response from `/api/products` and `/api/products/{id}` MUST have
  `Content-Type: application/problem+json` and a JSON body with `status` (the HTTP status as a
  number), `title` (the status's standard reason phrase), a non-empty `detail`, and `instance`
  (the request path, without the query string).
- **FR-002**: A request body that fails validation MUST answer `400`, store nothing, and add
  `errors`: an array with exactly one object per invalid field, `{"field": <name>, "message":
  <non-empty text>}`, sorted by `field`. Valid fields MUST NOT appear in it.
- **FR-003**: `GET /api/products/{id}` for an id that doesn't exist MUST answer `404` with a
  `detail` that contains the id.
- **FR-004**: A request body that isn't valid JSON, or an `{id}` that isn't a whole number,
  MUST answer `400` as a Problem Detail (FR-001); `errors` is optional there.
- **FR-005**: Successful responses MUST stay exactly as 002 defines them.
- **FR-006**: Tests for this item MUST be added under `src/test/java/dev/stockroom/`, and every
  existing test MUST keep passing unchanged. No dependency may be added.

## Allowed paths

- `src/main/java/dev/stockroom/*`
- `src/main/resources/*`
- `src/test/java/dev/stockroom/*`

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every acceptance scenario above passes as an automated test under `mvn test`.
- **SC-002**: The 001 and 002 tests pass without being edited.

## Assumptions

- No earlier test checks an error body, only status codes, so none needs to change.
- `type` is left at its RFC 9457 default (`about:blank`); per-error type URIs can come later.

# Plan

# Implementation Plan: Problem Details errors

**Branch**: `003-problem-details` | **Date**: 2026-11-18 | **Spec**: [spec.md](spec.md)

## Summary

A `@RestControllerAdvice` in `dev.stockroom` that extends Spring's
`ResponseEntityExceptionHandler`, so the framework's own errors (validation, unreadable body,
type mismatch) already come back as `ProblemDetail`. It adds the `errors` list to validation
failures, sets `instance` to the request path, and maps a new `ProductNotFoundException` to
`404`. The controller throws that exception instead of building an empty `404`.

## Technical Context

**Language/Version**: Java 21 | **Framework**: Spring Boot 4.1.2 (`webmvc`, `validation`)
**Testing**: JUnit 5, `@SpringBootTest` + MockMvc | **Build**: `mvn test`
**Constraints**: no new dependencies; tests offline; existing tests unchanged

## Constitution Check

I item 003, depends on 002 ✓ · II owner merges ✓ · III one PR with tests ✓ · IV no dependency
change ✓ · V offline tests ✓ · VI no planning files or CI touched ✓

## Project Structure

```text
src/main/java/dev/stockroom/ApiExceptionHandler.java
src/main/java/dev/stockroom/product/ProductNotFoundException.java
src/main/java/dev/stockroom/product/ProductController.java   (404 via the exception)
src/test/java/dev/stockroom/product/ProductErrorTests.java
```

# Task list (`specs/003-problem-details/tasks.md`)

# Tasks: Problem Details errors

**Input**: `specs/003-problem-details/` (spec.md, plan.md)

- [ ] T001 [US2] Add `ProductNotFoundException(long id)` in `src/main/java/dev/stockroom/product/`, and make `GET /api/products/{id}` in `ProductController` throw it when the product is missing (FR-003)
- [ ] T002 [US1] Add `ApiExceptionHandler` (`@RestControllerAdvice`, extends `ResponseEntityExceptionHandler`) in `src/main/java/dev/stockroom/`: `status`, `title`, non-empty `detail` and `instance` (request path) on every error; `errors` (one `{field, message}` per invalid field, sorted by `field`) on validation failures; `404` for `ProductNotFoundException` with the id in `detail` (FR-001, FR-002, FR-003)
- [ ] T003 [US3] Make sure malformed JSON and a non-numeric `{id}` answer `400` Problem Details through the same handler (FR-004)
- [ ] T004 Add tests in `src/test/java/dev/stockroom/product/ProductErrorTests.java` for every acceptance scenario of US1–US3, including `Content-Type: application/problem+json` (FR-006)
- [ ] T005 Run `mvn test` and confirm the 001 and 002 tests pass unedited (FR-005, SC-002)
