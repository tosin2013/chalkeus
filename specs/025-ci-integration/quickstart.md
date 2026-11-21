# Quickstart: integration tests

Locally (Docker, Go and git installed):

```text
go test -tags integration -timeout 45m ./test/integration/      # added by this feature
```

This builds the fake GitHub and the test images, stands up kind, runs every scenario and tears
it down. On failure, `test/integration/out/` holds each scenario's evidence and pod logs, as
the CI artifacts do.

To reproduce a live bug on purpose (SC-001): change the author pod label in
`pilot/manifests/author-job.yaml.tmpl` back to `app=worker`, and the automatic-drafting scenario
fails at A01.
