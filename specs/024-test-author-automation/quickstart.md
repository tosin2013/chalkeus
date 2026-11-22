# Quickstart: test-author automation

1. In the target's profile, set `PILOT_AUTHOR_AUTO=on`. Then run `pilot/pilot up`, which
   creates the author proxy.
2. In your own terminal, never the session that drives runs, run:

   ```text
   pilot/pilot author watch --target stockroom      # added by this feature
   ```
3. Plan a new item: merge its spec and tasks. Within a few minutes the watcher prints
   `draft NNN: started …`, then `DRAFT_OK, opened <companion PR url>`.
4. Review the PR as you would a manual draft, and merge it.
5. Run the item as usual:
   - `pilot/pilot doctor --target stockroom --item NNN --baseline`;
   - `pilot/pilot run --target stockroom --item NNN`;
   - after `down`, the audit should be 11/11 (SC-005).

**Check:** the draft's `draft.json` shows A01–A05 passing under `jobs[].checks` (SC-002), and
`trigger: auto`.
