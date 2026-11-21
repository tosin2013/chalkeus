You are a coding agent working in a git repository (the current directory).
Implement the specification below completely, then mark every task in the task list as
done by changing `- [ ]` to `- [X]` in that tasks.md file.

Rules:
- Change only files matching these paths (`*` also matches `/`):
  - `src/main/java/dev/stockroom/*`
  - `src/test/java/dev/stockroom/*`
  - `specs/009-delete-product/tasks.md`
- The project uses the jdk21-maven toolchain. Check your work with: `mvn -B test`
- Add or extend tests for your change in the project's test files (`src/test/*`). Keep every existing
  test passing unchanged: the grader re-runs the original versions of any test file you edit or delete.
- Only the project's existing dependencies are available to the grader; do not add new ones.
- Do not run git commands; the harness commits your work.
- When finished, stop.

# Specification

# Feature Specification: Delete a product

**Feature Branch**: `009-delete-product`

**Created**: 2026-11-19

**Status**: Draft

**Input**: Roadmap item 009. A client removes a product it no longer stocks. Depends on 003
(Problem Details errors) and 004 (persistence). Also the first item whose hidden tests are
drafted automatically (specfoundry 024).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Delete an existing product (Priority: P1)

As a client, I delete a product by its id, and it is gone from every read.

**Why this priority**: The catalogue can only grow today; a discontinued product stays forever.

**Independent Test**: Create a product, delete it, then read it and list the catalogue.

**Acceptance Scenarios**:

1. **Given** an existing product, **When** a client calls `DELETE /api/products/{id}`, **Then**
   the answer is `204` with an empty body.
2. **Given** that product was deleted, **When** a client calls `GET /api/products/{id}`, **Then**
   the answer is `404` as a Problem Detail (003), and `GET /api/products` no longer lists it.
3. **Given** two products, **When** one is deleted, **Then** the other is unchanged: same name and
   stock, still readable by its id and still listed.

---

### User Story 2 - Deleting what isn't there (Priority: P1)

As a client, deleting an unknown product tells me so, the same way reading one does.

**Why this priority**: A clear error stops a client from believing a delete worked.

**Independent Test**: Delete an id that never existed, and one already deleted.

**Acceptance Scenarios**:

1. **Given** no product with id `999999`, **When** a client calls `DELETE /api/products/999999`,
   **Then** the answer is `404` as a Problem Detail with `status` `404` and a `detail` naming
   the id.
2. **Given** a product that was just deleted, **When** a client deletes it again, **Then** the
   answer is `404`.
3. **Given** an id that isn't a whole number, **When** a client calls `DELETE /api/products/abc`,
   **Then** the answer is `400` as a Problem Detail (003 FR-004).

---

### User Story 3 - A deleted product can't be reserved (Priority: P2)

As a client, reserving a deleted product fails like reserving one that never existed.

**Why this priority**: Stock must not be taken from a product that no longer exists.

**Independent Test**: Delete a product, then reserve it.

**Acceptance Scenarios**:

1. **Given** a deleted product, **When** a client calls `POST /api/products/{id}/reservations`
   with a quantity of `1`, **Then** the answer is `404` as a Problem Detail, and no
   `StockReserved` event is published.

---

### Edge Cases

- Deleting doesn't renumber anything: other products keep their ids, and a new product never
  reuses a deleted one's id.
- Paging (006) counts only existing products after a delete: `totalElements` drops by one.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `DELETE /api/products/{id}` MUST remove the product and answer `204` with no body.
- **FR-002**: For an id with no product, it MUST answer `404` as a Problem Detail (003 FR-001,
  FR-003). An id that isn't a whole number MUST answer `400` (003 FR-004).
- **FR-003**: After a delete, every read (by id, the plain list, paging and filtering) and every
  reservation MUST behave as if the product never existed.
- **FR-004**: Everything else MUST stay the same (001-008).
- **FR-005**: Tests for this item MUST be added under `src/test/java/dev/stockroom/`, and every
  existing test MUST keep passing unedited. No dependency may be added.

## Allowed paths

- `src/main/java/dev/stockroom/*`
- `src/test/java/dev/stockroom/*`

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every acceptance scenario above passes as an automated test under `mvn test`, and
  fails on the 008 code.
- **SC-002**: The 001-008 tests pass without being edited.

## Assumptions

- A delete is permanent; there is no soft delete or undo.
- Reservations aren't stored (005), so nothing else references a product.
- Tests share one in-memory database per Spring context, so each test creates its own products.

# Plan

# Implementation Plan: Delete a product

**Branch**: `009-delete-product` | **Date**: 2026-11-19 | **Spec**: [spec.md](spec.md)

## Summary

Add `DELETE /api/products/{id}` to `ProductController`, backed by a new
`ProductStore.delete(long id)`. The store throws `ProductNotFoundException` when no row was
deleted, and the existing `ApiExceptionHandler` turns that into a `404` Problem Detail. Reads and
reservations already answer `404` for a missing row, so nothing else changes.

## Technical Context

**Language/Version**: Java 21 | **Framework**: Spring Boot 4.1.2 (`data-jdbc`, embedded H2)
**Testing**: JUnit 5, `@SpringBootTest` + MockMvc | **Build**: `mvn test`
**Constraints**: no new dependencies; tests offline; existing tests unchanged

## Constitution Check

I item 009, depends on 003 and 004 ✓ · II owner merges ✓ · III one PR with tests ✓ · IV no
dependency change ✓ · V offline tests ✓ · VI no planning files or CI touched ✓

## Project Structure

```text
src/main/java/dev/stockroom/product/ProductController.java   (DELETE /{id} → 204)
src/main/java/dev/stockroom/product/ProductStore.java        (delete: 404 when nothing deleted)
src/test/java/dev/stockroom/product/ProductDeleteTests.java
```

# Task list (`specs/009-delete-product/tasks.md`)

# Tasks: Delete a product

**Input**: `specs/009-delete-product/` (spec.md, plan.md)

- [ ] T001 [US1] [US2] Add `delete(long id)` to `src/main/java/dev/stockroom/product/ProductStore.java`: delete the row, and throw `ProductNotFoundException` when no row was deleted (FR-001, FR-002)
- [ ] T002 [US1] [US2] Add `DELETE /api/products/{id}` to `src/main/java/dev/stockroom/product/ProductController.java`, answering `204` with no body (FR-001)
- [ ] T003 Add tests in `src/test/java/dev/stockroom/product/ProductDeleteTests.java` for every acceptance scenario and edge case of US1–US3 (FR-003, FR-005)
- [ ] T004 Run `mvn test` and confirm the 001–008 tests pass unedited (FR-004, SC-002)
