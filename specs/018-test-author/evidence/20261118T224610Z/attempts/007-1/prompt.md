You are a coding agent working in a git repository (the current directory).
Implement the specification below completely, then mark every task in the task list as
done by changing `- [ ]` to `- [X]` in that tasks.md file.

Rules:
- Change only files matching these paths (`*` also matches `/`):
  - `src/main/java/dev/stockroom/*`
  - `src/test/java/dev/stockroom/*`
  - `specs/007-reservation-stock/tasks.md`
- The project uses the jdk21-maven toolchain. Check your work with: `mvn -B test`
- Add or extend tests for your change in the project's test files (`src/test/*`). Keep every existing
  test passing unchanged: the grader re-runs the original versions of any test file you edit or delete.
- Only the project's existing dependencies are available to the grader; do not add new ones.
- Do not run git commands; the harness commits your work.
- When finished, stop.

# Specification

# Feature Specification: Reservations report the stock they left

**Feature Branch**: `007-reservation-stock`

**Created**: 2026-11-18

**Status**: Draft

**Input**: Roadmap item 007, found in the owner's review of 005 (stockroom#25). Stock is never
oversold, but a reservation computes `remainingStock` (and a refusal's `available`) from a read
taken *before* its atomic update, so under concurrency it reports a stale figure: two
reservations of `1` against stock `10` both report `9` while the stock correctly ends at `8`.
Depends on 005.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Each reservation reports the stock it left (Priority: P1)

As a client, the `remainingStock` I get back is the stock right after my reservation, even when
other clients reserve at the same moment.

**Why this priority**: Clients and event listeners act on that figure; a stale one misleads them.

**Independent Test**: Many concurrent reservations; compare the reported figures.

**Acceptance Scenarios**:

1. **Given** a product with stock `100`, **When** 50 clients each reserve `1` at the same time,
   **Then** all 50 get `200`, the 50 `remainingStock` values are exactly the numbers `50` to `99`,
   each once, and the stock ends at `50`.
2. **Given** a product with stock `100`, **When** 30 clients reserve at the same time with
   quantities of `1`, `2` or `3`, **Then** every response's `remainingStock` differs, and sorting
   the successful responses by `remainingStock` descending, each one equals the one before it
   minus its own `quantity` (the first equals `100` minus its `quantity`).

---

### User Story 2 - Events carry the same figure (Priority: P1)

As a listener of `StockReserved`, the event's `remainingStock` matches the response's.

**Why this priority**: The event is how other parts of the system learn the stock.

**Independent Test**: Collect events from all threads during concurrent reservations.

**Acceptance Scenarios**:

1. **Given** the reservations of User Story 1 scenario 1, **When** every `StockReserved` event is
   collected (by a listener, whatever thread publishes it), **Then** there are 50, and their
   `remainingStock` values are exactly `50` to `99`, each once.

---

### User Story 3 - Refusals report what is really available (Priority: P2)

As a client refused with `409`, the `available` I get back is the stock after the attempts that
beat mine, not a figure read before them.

**Why this priority**: A client deciding how much to retry needs the real figure.

**Independent Test**: Exhaust the stock concurrently and read the refusals.

**Acceptance Scenarios**:

1. **Given** a product with stock `5`, **When** 15 clients each reserve `1` at the same time,
   **Then** exactly 5 get `200` and 10 get `409`, and every `409` reports `"available": 0`.

---

### Edge Cases

- Without concurrency nothing changes: every 005 scenario answers as before.
- The figures are the stock *as this reservation left it*; a later reservation may lower it
  again before the client reads the product.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A successful reservation's `remainingStock`, in the response and in its
  `StockReserved` event, MUST be the product's stock immediately after that reservation's own
  decrement, read in the same transaction after the update (or returned by it), never computed
  from a read taken before it.
- **FR-002**: A refused reservation's `available` MUST be the product's stock read after the
  failed conditional update, not before it.
- **FR-003**: Everything else about reservations (005) MUST stay the same: atomic decrement, no
  overselling, status codes, bodies, one event per success and none per refusal.
- **FR-004**: Tests for this item MUST be added under `src/test/java/dev/stockroom/`, and every
  existing test MUST keep passing unedited. No dependency may be added.

## Allowed paths

- `src/main/java/dev/stockroom/*`
- `src/test/java/dev/stockroom/*`

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every acceptance scenario above passes as an automated test under `mvn test`, and
  fails on the 005 code.
- **SC-002**: The 001–005 tests pass without being edited.

## Assumptions

- The embedded H2 database locks a row from its update until the transaction commits, so a read
  in the same transaction right after the update sees exactly this reservation's result.
- Tests share one in-memory database per Spring context; each test creates its own products.

# Plan

# Implementation Plan: Reservations report the stock they left

**Branch**: `007-reservation-stock` | **Date**: 2026-11-18 | **Spec**: [spec.md](spec.md)

## Summary

In `ProductStore.reserve` (already `@Transactional`), run the conditional decrement first. If it
updated a row, read the product's stock again in the same transaction and use that figure for
the `Reservation` and the `StockReserved` event. If it updated nothing, read the product: missing
→ `404`; present → `409` with that freshly read stock as `available`. Drop the read taken before
the update.

## Technical Context

**Language/Version**: Java 21 | **Framework**: Spring Boot 4.1.2 (`data-jdbc`, embedded H2)
**Testing**: JUnit 5, `@SpringBootTest` + MockMvc, an `ExecutorService` for concurrency, a test
`@EventListener` that collects events from every thread | **Build**: `mvn test`
**Constraints**: no new dependencies; tests offline; existing tests unchanged

## Constitution Check

I item 007, depends on 005 ✓ · II owner merges ✓ · III one PR with tests ✓ · IV no dependency
change ✓ · V offline tests ✓ · VI no planning files or CI touched ✓

## Project Structure

```text
src/main/java/dev/stockroom/product/ProductStore.java   (reserve: update, then read)
src/test/java/dev/stockroom/product/ReservationConcurrencyTests.java
```

# Task list (`specs/007-reservation-stock/tasks.md`)

# Tasks: Reservations report the stock they left

**Input**: `specs/007-reservation-stock/` (spec.md, plan.md)

- [ ] T001 [US1] In `src/main/java/dev/stockroom/product/ProductStore.java`, make `reserve` run the conditional decrement first and then read the stock in the same transaction; use that figure for `Reservation.remainingStock` and `StockReserved.remainingStock` (FR-001)
- [ ] T002 [US3] In `src/main/java/dev/stockroom/product/ProductStore.java`, when the decrement updates nothing, read the product then: missing → `ProductNotFoundException`; present → `InsufficientStockException` with that stock as `available` (FR-002)
- [ ] T003 Add tests in `src/test/java/dev/stockroom/product/ReservationConcurrencyTests.java` for every acceptance scenario of US1–US3, collecting `StockReserved` events from all threads (FR-004)
- [ ] T004 Run `mvn test` and confirm the 001–005 tests pass unedited (FR-003, SC-002)
