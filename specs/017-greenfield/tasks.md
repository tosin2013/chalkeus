# Tasks: Greenfield bootstrap

**Input**: `specs/017-greenfield/spec.md`. Written after the work, as its record: 017 was built
guide-first on a real repository, Stockroom (`tosin2013/stockroom`), so each task names what was
done and where.

- [X] T001 Plan Stockroom in one planning PR (constitution, roadmap, specs 001 and 002, `pom.xml`, CI): stockroom#1 (FR-002, FR-005)
- [X] T002 Add `pilot-maven-prefetch` to the `jdk21-maven` toolchain image, with a sample whose hidden tests use a class that doesn't exist yet: `pilot/images/toolchain-jdk21-maven/` (FR-003)
- [X] T003 Write `docs/greenfield.md`, from an empty repository to a verified first item, linked from `docs/targets.md` (FR-006)
- [X] T004 Run the scaffold (001) to `verified`: run 20261118T161914Z (FR-004, SC-001, SC-002)
- [X] T005 Run a dependent item (002) to `verified` after a planning fix that declares the 001 test it changes (`## Tests that may change`, check G7): stockroom#11, run 20261118T171710Z (FR-007, SC-003)
- [X] T006 Run the rest of Stockroom's roadmap (003–007) to `verified`, all on their first attempt; 007 came from the owner's review of 005: runs in `specs/018-test-author/evidence/`
- [X] T007 Tear down and audit: the last run, 20261118T230306Z, audits 11/11 after `down` (SC-005)
- [X] T008 Drop the Node.js greenfield (US5, FR-008, SC-004): owner decision 2026-11-18, Stockroom is the reference
