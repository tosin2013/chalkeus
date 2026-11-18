You are a coding agent working in a git repository (the current directory).
Implement the specification below completely, then mark every task in the task list as
done by changing `- [ ]` to `- [X]` in that tasks.md file.

Rules:
- Change only files matching these paths (`*` also matches `/`):
  - `src/main/java/dev/stockroom/*`
  - `src/main/resources/*`
  - `src/test/java/dev/stockroom/*`
  - `specs/004-persistence/tasks.md`
- The project uses the jdk21-maven toolchain. Check your work with: `mvn -B test`
- Add or extend tests for your change in the project's test files (`src/test/*`). Keep every existing
  test passing unchanged: the grader re-runs the original versions of any test file you edit or delete.
- Only the project's existing dependencies are available to the grader; do not add new ones.
- Do not run git commands; the harness commits your work.
- When finished, stop.

# Specification

# Feature Specification: Persistence

**Feature Branch**: `004-persistence`

**Created**: 2026-11-18

**Status**: Draft

**Input**: Roadmap item 004. Products are stored in the embedded H2 database through Spring Data
JDBC, with a schema; the products API behaves exactly as before. Depends on 003.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Products live in a database table (Priority: P1)

As the owner, I want products stored in a real table with a schema, so later items
(reservations, paging) can query and update them safely.

**Why this priority**: The in-memory store can't enforce rules, lock rows or page results.

**Independent Test**: Create a product through the API, then read the `product` table directly.

**Acceptance Scenarios**:

1. **Given** the application has started, **When** the database is inspected, **Then** a table
   `product` exists with columns `id`, `name` and `stock`.
2. **Given** a client posts `{"name": "Widget", "stock": 5}`, **When** the `product` table is
   queried for the returned id, **Then** it holds one row with `name` `Widget` and `stock` `5`.
3. **Given** a row inserted directly into `product` (`name` `Bolt`, `stock` `3`), **When** a
   client calls `GET /api/products/{its id}` and `GET /api/products`, **Then** both return it.

---

### User Story 2 - The database enforces the rules too (Priority: P2)

As the owner, I want the table to refuse data the API would refuse, so a bug in one place can't
corrupt the catalogue.

**Why this priority**: Defence in depth for the data later items depend on.

**Independent Test**: Insert bad rows directly and expect the database to refuse them.

**Acceptance Scenarios**:

1. **Given** the `product` table, **When** a row with a `NULL` name, a name of 101 characters,
   or a negative `stock` is inserted directly, **Then** the database refuses it.
2. **Given** a row inserted without an `id`, **When** it is read back, **Then** the database
   assigned a positive id greater than every earlier one.

---

### User Story 3 - Nothing changes for clients (Priority: P1)

As a client, I see the same API as before: same status codes, bodies, ordering and errors.

**Why this priority**: This item changes storage, not behaviour.

**Independent Test**: The 002 and 003 tests pass unedited.

**Acceptance Scenarios**:

1. **Given** products created in an order, **When** a client calls `GET /api/products`, **Then**
   they come back in creation order (ascending id), as in 002.
2. **Given** any request covered by 002 or 003, **When** it is made, **Then** the response is
   the same as before this item.

---

### Edge Cases

- Concurrent creations each get a distinct id (the database assigns them).
- The database is the in-memory embedded H2 the planning PR declared; data does not survive a
  restart, and no file or server database is configured.
- The H2 web console stays off.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The application MUST create a table `product` at startup from a schema file under
  `src/main/resources/`, with `id` (a whole number the database assigns, primary key), `name`
  (text, not null, at most 100 characters) and `stock` (whole number, not null, `>= 0`).
- **FR-002**: Products MUST be read and written through a Spring Data JDBC repository interface
  `ProductRepository` in the package `dev.stockroom.product`.
- **FR-003**: `ProductStore` MUST remain a bean with its public methods `save(String name, int
  stock)`, `findById(long id)` and `findAll()` and their return types, now backed by
  `ProductRepository`; `findAll()` MUST return products in ascending id order.
- **FR-004**: The products API (002) and its errors (003) MUST behave exactly as before.
- **FR-005**: The application MUST use the embedded in-memory H2 database; no other datasource,
  and the H2 console off.
- **FR-006**: Tests for this item MUST be added under `src/test/java/dev/stockroom/`, including
  tests of the repository, and every existing test MUST keep passing unedited. No dependency
  may be added.

## Allowed paths

- `src/main/java/dev/stockroom/*`
- `src/main/resources/*`
- `src/test/java/dev/stockroom/*`

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every acceptance scenario above passes as an automated test under `mvn test`.
- **SC-002**: The 001, 002 and 003 tests pass without being edited.

## Assumptions

- Existing tests inject `ProductStore` and call `findAll()`; keeping its methods (FR-003) means
  no existing test needs to change, so this spec has no `## Tests that may change`.
- Tests share one in-memory database per Spring context, so tests must not assume an empty
  table unless they clear it themselves.

# Plan

# Implementation Plan: Persistence

**Branch**: `004-persistence` | **Date**: 2026-11-18 | **Spec**: [spec.md](spec.md)

## Summary

A `schema.sql` (run by Spring Boot's SQL initialisation for the embedded database) creates
`product` with an identity `id`, `name VARCHAR(100) NOT NULL` and `stock INT NOT NULL CHECK
(stock >= 0)`. `ProductRepository extends ListCrudRepository<…, Long>` maps it with Spring Data
JDBC, either on the `Product` record (`@Id`, `@Table("product")`) or on a separate row record.
`ProductStore` keeps its public methods and delegates to the repository; `findAll()` sorts by
id. The controller, the 003 error handler and the existing tests don't change.

## Technical Context

**Language/Version**: Java 21 | **Framework**: Spring Boot 4.1.2 (`data-jdbc`, embedded H2)
**Testing**: JUnit 5, `@SpringBootTest` + MockMvc, `JdbcTemplate` for direct table checks,
`@DataJdbcTest` for the repository | **Build**: `mvn test`
**Constraints**: no new dependencies; tests offline; existing tests unchanged

## Constitution Check

I item 004, depends on 003 ✓ · II owner merges ✓ · III one PR with tests ✓ · IV no dependency
change (data-jdbc and H2 declared in planning) ✓ · V offline tests (in-memory H2) ✓ · VI no
planning files or CI touched ✓

## Project Structure

```text
src/main/resources/schema.sql
src/main/java/dev/stockroom/product/ProductRepository.java
src/main/java/dev/stockroom/product/Product.java        (mapping annotations, if mapped here)
src/main/java/dev/stockroom/product/ProductStore.java   (delegates to the repository)
src/test/java/dev/stockroom/product/ProductRepositoryTests.java
src/test/java/dev/stockroom/product/ProductPersistenceTests.java
```

# Task list (`specs/004-persistence/tasks.md`)

# Tasks: Persistence

**Input**: `specs/004-persistence/` (spec.md, plan.md)

- [ ] T001 [US1] Add `src/main/resources/schema.sql` creating table `product`: `id` identity primary key, `name VARCHAR(100) NOT NULL`, `stock INT NOT NULL CHECK (stock >= 0)` (FR-001)
- [ ] T002 [US1] Add `ProductRepository` (Spring Data JDBC, `ListCrudRepository`) in `src/main/java/dev/stockroom/product/`, with the mapping for table `product` (FR-002)
- [ ] T003 [US3] Back `ProductStore` in `src/main/java/dev/stockroom/product/ProductStore.java` with `ProductRepository`, keeping `save(String, int)`, `findById(long)` and `findAll()` and their return types; `findAll()` in ascending id order (FR-003, FR-004)
- [ ] T004 [US2] Add repository tests in `src/test/java/dev/stockroom/product/ProductRepositoryTests.java`: save and find, ids increase, and the table refuses a null name, a 101-character name and negative stock (FR-001, FR-006)
- [ ] T005 [US1] Add tests in `src/test/java/dev/stockroom/product/ProductPersistenceTests.java`: a product posted through the API is a row in `product`, and a row inserted directly is returned by the API (FR-002, FR-004)
- [ ] T006 Run `mvn test` and confirm the 001, 002 and 003 tests pass unedited (FR-004, SC-002)
