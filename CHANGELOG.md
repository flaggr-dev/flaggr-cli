# Changelog

## Unreleased

FEATURES:

* `flaggr sourcemaps upload|list|delete`: upload a release's source maps (`--release`, `--url-prefix`, `--dry-run`), gzipped and four at a time. Flaggr's Issues page then shows minified stack frames as your original source.
* Opt-in crash reports: `flaggr crash-reports on|off|status` (or `FLAGGR_CRASH_REPORTS=1|0`). When they're on, a panic is reported to the Flaggr team with its message, stack, the CLI's version, the subcommand name and the OS, architecture and Go version. The report never includes the token, arguments or output. `DO_NOT_TRACK=1` and CI environments keep reports off. The panic still prints and exits as before. Reports use the Go SDK's new `errorreport` package.

## 0.5.0 (October 10, 2026)

First public release, from `github.com/flaggr-dev/flaggr-cli`. Earlier versions were built in Flaggr's private repository and were available only on request.

FEATURES:

* Install with the install script (`install.sh`, for macOS and Linux), from a release archive, or with `go install github.com/flaggr-dev/flaggr-cli/cmd/flaggr@latest`. Releases add Windows archives to the macOS and Linux ones, for amd64 and arm64, with `checksums.txt`; the install script checks the download against it.
* `flaggr --version` (and `flaggr status`) report the release version, or, for a `go install` build, the module version, with the commit when Go records one or the version is a pseudo-version (`@main`, or `@latest` before a release).
* Commands: `login`, `logout`, `status`, `projects`, `services`, `flags` (`list`, `create`, `toggle`, `delete`, `stale`), `eval` (`bool`, `string`, `number`, `config` and `stream`, over Connect-RPC), `metrics` (`flag`, `watch`, `heatmap`), `health` (`flag`, `project`), `audit` (`log`, `history`) and `export`. Every command takes `--json`.

SECURITY:

* `flaggr login` sends a one-time `state` to the sign-in page and accepts only the callback that returns it, so another web page can't log the CLI into a different account through its localhost port.
* The sign-in page hands the token to the CLI's localhost callback in a form POST, never in a URL, where browser history would keep it. A token sent in the callback URL is refused and not saved: revoke it in Flaggr.
* A sign-in page from before Flaggr 0.5.0 puts its token in the callback URL without a `state`. Since any web page can send that request, it doesn't end the login: the CLI keeps waiting, and says once, in the terminal, which token to revoke and to use `flaggr login --token`.
* `flaggr logout` says the token stays valid until it expires or you revoke it, and where to revoke it, and warns while `FLAGGR_API_TOKEN` is set.

NOTES:

* Browser login needs a Flaggr server from release 0.5.0 or later; with an older self-hosted server, use `flaggr login --token`. CLI releases before 0.5.0 can't log in through the browser to flaggr.dev: update, or use `flaggr login --token`.
* Earlier browser logins passed their token through a URL: consider revoking old "CLI login (date)" tokens.
* Release archives include `THIRD_PARTY_NOTICES`, the licenses of Go and of the modules compiled into `flaggr`.
* `flaggr login` → **One project** creates a read & write project API token, so `flaggr flags delete` gets 403 with it. Choose **All my projects** (a personal access token), or pass a token with the Delete permission to `flaggr login --token`.
