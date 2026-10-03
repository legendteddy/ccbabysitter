# Contributing to CC Babysitter

Thanks for helping. This page says how to build and test CC Babysitter and how changes get in.

## Build and test

You need Go 1.27 or later. The page tests in `internal/web` run the page's JavaScript under `node` and are skipped when it is missing, so install Node.js too if you change anything under `internal/web`.

```
go build ./...
go vet ./...
go test ./...
gofmt -l .
```

`gofmt -l .` must print nothing. CI also runs the tests with `-race` on macOS and Linux.

To try the page without touching any real session, run `go run ./cmd/ccbabysitter --demo`.

If you change `scripts/install.sh`, run `scripts/test-install.sh`. It builds CC Babysitter, serves the release files from this machine, and runs the script against them with a temporary home folder. It needs `go`, `python3`, `curl`, and `sha256sum` or `shasum`.

## Sending a change

- `main` is protected. Every change comes as a pull request, and it is merged only when CI is green: `go vet`, the tests on Linux, macOS and Windows, and the `gofmt` check.
- Keep pull requests small and focused on one thing. A small change is easier to review and quicker to merge.
- For a larger change, or anything that changes what CC Babysitter does, open an issue first so we can agree on the approach before you spend time on it.
- Keep to the hard rules in `README.md`: opt-in only, every automatic action explained, Claude's files read-only, loopback only, user mode only.
- Shipped files are plain ASCII text: no curly quotes, long dashes or other characters outside ASCII. `go test ./...` checks this for the code, the documents, the scripts and the workflows.

## Security problems

Do not open a public issue for a vulnerability. See `SECURITY.md`.

## License

By contributing, you agree that your contribution is licensed under the MIT license in `LICENSE`.
