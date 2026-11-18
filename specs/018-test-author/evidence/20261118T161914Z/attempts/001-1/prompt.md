You are a coding agent working in a git repository (the current directory).
Implement the specification below completely, then mark every task in the task list as
done by changing `- [ ]` to `- [X]` in that tasks.md file.

Rules:
- Change only files matching these paths (`*` also matches `/`):
  - `src/main/java/dev/stockroom/*`
  - `src/main/resources/*`
  - `src/test/java/dev/stockroom/*`
  - `specs/001-scaffold/tasks.md`
- The project uses the jdk21-maven toolchain. Check your work with: `mvn -B test`
- Add or extend tests for your change in the project's test files (`src/test/*`). Keep every existing
  test passing unchanged: the grader re-runs the original versions of any test file you edit or delete.
- Only the project's existing dependencies are available to the grader; do not add new ones.
- Do not run git commands; the harness commits your work.
- When finished, stop.

# Specification

# Feature Specification: Scaffold

**Feature Branch**: `001-scaffold`

**Created**: 2026-11-18

**Status**: Draft

**Input**: Roadmap item 001. The empty repository becomes a Spring Boot application that starts,
reports its health and has its first test. Dependencies and CI already exist from the planning
pull request.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - The service starts and reports its health (Priority: P1)

As an operator, I can start Stockroom and ask whether it is healthy, so that I can run it and
monitor it from day one.

**Why this priority**: Every later item builds on a running application.

**Independent Test**: Start the application in a test and call `GET /actuator/health`.

**Acceptance Scenarios**:

1. **Given** the application, **When** its Spring context starts in a test, **Then** it starts
   without errors.
2. **Given** the running application, **When** a client calls `GET /actuator/health`, **Then**
   the response is `200` with a JSON body whose `status` is `UP`.
3. **Given** the running application, **When** a client calls `GET /actuator/info` or
   `GET /actuator/env`, **Then** the response is `404`: only the health endpoint is exposed.

---

### User Story 2 - The project is ready to grow (Priority: P1)

As a developer, I find a conventional layout, virtual threads on, and a passing test, so that
the next roadmap item starts from a working base.

**Why this priority**: The scaffold is only useful if the next item can build on it.

**Independent Test**: Read the configuration and run `mvn test`.

**Acceptance Scenarios**:

1. **Given** the application's configuration, **When** it is read, **Then**
   `spring.threads.virtual.enabled` is `true`.
2. **Given** the project, **When** `mvn test` runs, **Then** at least one test of the project's
   own passes.

---

### Edge Cases

- The application must start without a database configured explicitly: embedded H2 is on the
  classpath (004 uses it), and starting must not fail because of it.
- No business endpoints exist yet: `GET /api/products` returns `404` until 002.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The application class MUST be `dev.stockroom.StockroomApplication`, annotated
  `@SpringBootApplication`, with a `main` method that starts it.
- **FR-002**: `GET /actuator/health` MUST return `200` and a JSON body with `"status": "UP"`.
- **FR-003**: Only the `health` actuator endpoint MUST be exposed over HTTP
  (`management.endpoints.web.exposure.include=health`).
- **FR-004**: `src/main/resources/application.properties` MUST set
  `spring.threads.virtual.enabled=true` and `spring.application.name=stockroom`.
- **FR-005**: The project MUST include at least one test of its own under
  `src/test/java/dev/stockroom/`, and `mvn test` MUST pass.
- **FR-006**: No dependency may be added or changed (`pom.xml` is outside this item).

## Allowed paths

- `src/main/java/dev/stockroom/*`
- `src/main/resources/*`
- `src/test/java/dev/stockroom/*`

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `mvn test` passes on a clean checkout with no network beyond the approved
  dependencies.
- **SC-002**: The health endpoint answers `UP` within the test's application start.

## Assumptions

- Java 21 and the dependencies in `pom.xml` (approved in the planning pull request).

# Plan

# Implementation Plan: Scaffold

**Branch**: `001-scaffold` | **Date**: 2026-11-18 | **Spec**: [spec.md](spec.md)

## Summary

Add the Spring Boot application class, `application.properties` (name, virtual threads, health
only), and a first test that starts the context and checks the health endpoint with MockMvc.

## Technical Context

**Language/Version**: Java 21 | **Framework**: Spring Boot 4.1.2 (parent in `pom.xml`)
**Testing**: JUnit 5, `@SpringBootTest` with `@AutoConfigureMockMvc` | **Build**: Maven (`mvn test`)
**Constraints**: no new dependencies; tests offline (constitution IV, V)

## Constitution Check

I roadmap item 001 ✓ · II owner merges ✓ · III one PR with tests ✓ · IV no dependency change ✓ ·
V offline tests ✓ · VI no planning files or CI touched ✓

## Project Structure

```text
src/main/java/dev/stockroom/StockroomApplication.java
src/main/resources/application.properties
src/test/java/dev/stockroom/StockroomApplicationTests.java
```

# Task list (`specs/001-scaffold/tasks.md`)

# Tasks: Scaffold

**Input**: `specs/001-scaffold/` (spec.md, plan.md)

- [ ] T001 [US1] Create `src/main/java/dev/stockroom/StockroomApplication.java` (`@SpringBootApplication`, `main`) (FR-001)
- [ ] T002 [US1] Create `src/main/resources/application.properties`: `spring.application.name=stockroom`, `spring.threads.virtual.enabled=true`, `management.endpoints.web.exposure.include=health` (FR-003, FR-004)
- [ ] T003 [US1] Add `src/test/java/dev/stockroom/StockroomApplicationTests.java`: the context starts; `GET /actuator/health` returns `200` with `status` `UP` (FR-002, FR-005)
- [ ] T004 [US2] Run `mvn test` and confirm it passes (SC-001)
