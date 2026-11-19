# Quickstart: step mode

1. In the target's profile, set `PILOT_GITHUB_ISSUES=on` and `PILOT_GITHUB_APPROVALS=on`, and
   set `PILOT_APPROVERS` in `pilot.env`. Each manager pass syncs the item issues (004's sync, as
   a run does), so a newly planned item gets its issue to label.
2. In your own terminal, run:

   ```text
   pilot/pilot step --target stockroom      # added by this feature
   ```
3. On GitHub, add the "pilot/go" label to item 009's issue.
4. Within a pass or two:
   - the manager runs `doctor --baseline` and comments the result;
   - it starts the run, whose tracking issue asks for `/approve mode`.

   Answer every gate on GitHub, and type no pilot command.
5. Afterwards: `down`, then `audit` should give 11/11 (SC-001). `step/stockroom/decisions.jsonl`
   shows each step.
