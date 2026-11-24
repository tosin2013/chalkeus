# Contract: capacity check and messages

Run by `forge doctor --target <t>` (area `== capacity`), and before staging by `forge run` and
`forge author check` (a fail refuses to start, exit 2, code `CAPACITY_INSUFFICIENT`).

## Pass

```
== capacity
  PASS capacity: largest role grader needs 4 CPU / 4Gi; node chalkeus-control-plane has 6.5 CPU / 11.2Gi free for it
```

## Fail: local cluster (kube context `kind-<cluster>`)

```
  FAIL CAPACITY_INSUFFICIENT: grader needs 4 CPU / 4Gi (FORGE_GRADER_CPU=4); this machine's cluster can give 3.5 CPU / 5.2Gi.
       Give your container runtime more CPU (Docker Desktop: Settings → Resources), or lower FORGE_GRADER_CPU in targets/documcp.env (floor 2).
```

## Fail: remote cluster

```
  FAIL CAPACITY_INSUFFICIENT: worker needs 8 CPU / 16Gi; the largest schedulable node has 7.6 CPU / 30Gi allocatable after Chalkeus's pods.
       Use a larger node size or node pool, or lower FORGE_WORKER_CPU (floor 2).
```

## Fail: quota or limit range

```
  FAIL CAPACITY_INSUFFICIENT: ResourceQuota "compute" in pilot-g-* allows limits.cpu 2; grader needs 4.
       Raise the quota for the run namespaces, or lower FORGE_GRADER_CPU (floor 2).
```

## Warn: unreadable

```
  WARN capacity: could not check (nodes: forbidden); a pod that can't be scheduled will stop the run after 2 minutes
```

## Run-time stop (FR-011)

```
stopped: CAPACITY_UNSCHEDULABLE: grader pod waited 120 s: 0/1 nodes are available: 1 Insufficient cpu.
  grader needs 4 CPU / 4Gi. <same "what to change" line as above>
```

Recorded as event `capacity_stop`; the attempt isn't counted and the run ends `stopped`.
