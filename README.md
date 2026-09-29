# Electronic Tariff report retention

This scheduled Go Lambda removes old report objects from S3, including Electronic
Tariff files. It lists the configured UK and XI report prefixes and selects
objects by their last-modified date and retention period.

Source is in [electronic-tariff-file-rotations/](electronic-tariff-file-rotations/).
[serverless.yml](serverless.yml) defines the schedule and deployment settings.

## Develop and check changes

Use a Go version compatible with
[electronic-tariff-file-rotations/go.mod](electronic-tariff-file-rotations/go.mod).
From the repository root:

```sh
make build
make lint
```

The lint target requires golangci-lint. There is no Go test suite in this
repository. Do not run the compiled binary as a substitute for a test: outside
Lambda, `main` invokes the handler directly.

## Configuration and safety

- `ETF_BUCKET` selects the bucket.
- `S3_PREFIX` is a comma-separated list of prefixes. The defaults cover UK and XI reports.
- `DELETION_CANDIDATE_DAYS` sets the retention period.
- `DEBUG=true` logs candidates without deleting them. With debug disabled, the handler deletes candidates.

Even debug mode makes AWS requests. Confirm the account, bucket, prefixes and
retention period before any authorised operational run. Do not assume a local
invocation is harmless.

## Deployment

See the [Makefile](Makefile) and [deployment workflows](.github/workflows/).
A non-main branch push can deploy to shared development. Deployment is not
needed to review documentation or build the binary. Deletion and production
deployments require explicit approval.

## Contribute

Read [CONTRIBUTING.md](CONTRIBUTING.md) for the fork workflow, checks and private
security reporting.

## Licence

The repository uses the [MIT licence](LICENSE). Preserve its copyright notice.
The reports and third-party dependencies retain their own terms.
