# Quickstart: checking 027

## Each stage

```sh
go vet ./... && go test ./...        # includes the command-audit test and the docs check
go test -tags integration ./test/integration/   # the scenarios, with the current command form
```

- **Stage 1 (command audit):** in Claude Code, ask the agent to run `pilot/pilot down`. The harness
  asks first. Do the same for `reset-fixture`, `run` and `resume`. `pilot/pilot status` runs
  without asking.
- **Stage 2 (config):** with only `~/.config/specfoundry`, `pilot/pilot doctor` works and prints the
  move notice once. After `mv ~/.config/specfoundry ~/.config/chalkeus`, there's no notice.
  Setting `FORGE_MODEL` overrides `PILOT_MODEL`, with a notice.
- **Stage 3 (rename):** `./chalkeus forge doctor` works, `pilot/pilot` is gone, and the ask rules
  fire for `./chalkeus forge approve`. A fixture run passes in the integration job.
- **Stage 4 (cluster, labels, images):**
  - `./chalkeus forge doctor` names an old `specfoundry-m1` cluster and how to remove it;
  - after `down` and `up`, the cluster is `chalkeus-m1`;
  - the committed evidence audits row for row as before;
  - the image bot's PR pins `chalkeus-*`.

## The owner's rename day (after stage 3 merges)

1. GitHub → repository Settings → rename to `chalkeus`.
2. On each machine (optional; GitHub redirects):
   `git remote set-url origin https://github.com/tosin2013/chalkeus.git`.
3. Check: the go, integration and image workflows pass on the renamed repository.
