# Flaggr CLI

`flaggr` manages [Flaggr](https://flaggr.dev) feature flags from your terminal: sign in through your browser, list, create, toggle and delete flags, evaluate them over Connect-RPC, follow flag updates as they happen, and read evaluation metrics, health scores, the audit log and a project's exported configuration.

Documentation: [flaggr.dev/docs/getting-started/cli](https://flaggr.dev/docs/getting-started/cli)

## Install

### Install script (macOS and Linux)

```sh
curl -fsSL https://raw.githubusercontent.com/flaggr-dev/flaggr-cli/main/install.sh | sh
```

The script downloads the latest [release](https://github.com/flaggr-dev/flaggr-cli/releases) for your OS and architecture, checks it against the release's `checksums.txt`, and installs `flaggr` to `/usr/local/bin`, with `sudo` when that directory isn't writable. Set `FLAGGR_INSTALL_DIR` to install somewhere else, and `FLAGGR_VERSION` to install a given release:

```sh
curl -fsSL https://raw.githubusercontent.com/flaggr-dev/flaggr-cli/main/install.sh | FLAGGR_INSTALL_DIR="$HOME/.local/bin" FLAGGR_VERSION=0.5.0 sh
```

### Release downloads

Each [release](https://github.com/flaggr-dev/flaggr-cli/releases) has an archive for macOS (`darwin`), Linux (`linux`) and Windows (`windows`, a `.zip`), on `amd64` and `arm64`, named `flaggr_<version>_<os>_<arch>`, and `checksums.txt` with their SHA-256 sums. For example, on an Apple Silicon Mac:

```sh
curl -fsSLO https://github.com/flaggr-dev/flaggr-cli/releases/download/v0.5.0/flaggr_0.5.0_darwin_arm64.tar.gz
curl -fsSLO https://github.com/flaggr-dev/flaggr-cli/releases/download/v0.5.0/checksums.txt
shasum -a 256 --check --ignore-missing checksums.txt   # on Linux: sha256sum --check --ignore-missing checksums.txt
tar -xzf flaggr_0.5.0_darwin_arm64.tar.gz flaggr
sudo mv flaggr /usr/local/bin/
```

### With Go

With Go 1.26 or later:

```sh
go install github.com/flaggr-dev/flaggr-cli/cmd/flaggr@latest
```

This builds `flaggr` into `$(go env GOPATH)/bin`. `flaggr --version` reports the module version you installed.

## Get started

```sh
flaggr login                                   # opens your browser to sign in
flaggr status                                  # CLI and server versions, API health
flaggr projects                                # the projects you can access
flaggr flags list --project <project-id>
flaggr eval bool --key checkout-v2 --service <service-id>
```

## Signing in

`flaggr login` opens your browser at flaggr.dev. Once you're signed in, choose **One project**, a project API token (read & write) for the project you pick, or **All my projects**, a [personal access token](https://flaggr.dev/docs/api/personal-access-tokens) that acts as you in every project you can access. The sign-in page posts the token to the CLI's callback on `127.0.0.1`, never in a URL, along with a one-time `state` the CLI checks. The CLI saves the token in `~/.flaggr/config.json`, readable only by you.

Browser sign-in and personal access tokens need Flaggr 0.5.0 or later on the server: `flaggr status` shows its version. An older sign-in page creates a project API token named "CLI login (*date*)" and puts it in the callback URL, where your browser history keeps it, so `flaggr login` doesn't save it and tells you so in your terminal. Revoke that token in the project's **Settings → API tokens**, then create a project API token there and run `flaggr login --token fgr_...`.

A **One project** token can't delete: `flaggr flags delete` gets 403 with it. For deletes, choose **All my projects**, or pass a token that has the Delete permission to `flaggr login --token`.

In CI and other places without a browser, pass a token, or set `FLAGGR_API_TOKEN`, which takes precedence over the saved token:

```sh
flaggr login --token fgp_your_personal_access_token   # or a project API token (fgr_...)
```

`flaggr eval` calls the SDK evaluation endpoints, which accept project API tokens only. With a personal access token saved, pass a project token for the call: `FLAGGR_API_TOKEN=fgr_your_project_token flaggr eval bool --key checkout-v2 --service <service-id>`.

`flaggr logout` deletes `~/.flaggr/config.json`. It doesn't revoke the token: a personal access token works until it expires or you revoke it under **Profile → Personal access tokens**, and a project API token until you revoke it in the project's **Settings → API tokens**.

## Commands

| Command | What it does |
| --- | --- |
| `flaggr login`, `flaggr logout` | Sign in, or delete the saved credentials |
| `flaggr status` | CLI and server versions, and the API's health |
| `flaggr projects` | List the projects you can access |
| `flaggr services --project <id>` | List a project's services |
| `flaggr flags list --project <id>` | List flags, optionally only a `--service`'s or an `--environment`'s |
| `flaggr flags create --key <key> --name <name> --service <id>` | Create a flag: `--type` (`boolean`, `string`, `number` or `object`), `--default`, `--enabled` |
| `flaggr flags toggle --key <key> --service <id>` | Turn a flag on or off |
| `flaggr flags delete --key <key> --service <id>` | Delete a flag |
| `flaggr flags stale --project <id>` | Flags that haven't been evaluated recently |
| `flaggr eval bool --key <key> --service <id>` | Evaluate a flag over Connect-RPC (also `eval string` and `eval number`), with `--context '{"targetingKey":"user-123"}'` |
| `flaggr eval config --service <id>` | A service's flag configuration |
| `flaggr eval stream --service <id>` | Follow flag updates as they happen |
| `flaggr metrics flag --key <key> --service <id>` | A flag's evaluations, errors and latency over a `--range` |
| `flaggr metrics watch --key <key> --service <id>` | A flag's live evaluation rate |
| `flaggr metrics heatmap --project <id>` | Evaluations across a project's flags |
| `flaggr health flag --key <key> --service <id>` | A flag's health score |
| `flaggr health project --project <id>` | A project's health |
| `flaggr audit log --project <id>` | The project's audit log, with diffs |
| `flaggr audit history --key <key> --service <id>` | A flag's version history |
| `flaggr export --project <id>` | A project's configuration as JSON, to stdout or `--output <file>` |

Every command takes `--json` for JSON output. `flaggr <command> --help` lists a command's flags.

## Configuration

`~/.flaggr/config.json` holds the API URL, the token and the default project `flaggr login` saved. These environment variables override it:

| Variable | Default | What it sets |
| --- | --- | --- |
| `FLAGGR_API_URL` | `https://flaggr.dev` | The Flaggr API. For a self-hosted Flaggr, `flaggr login --url https://flaggr.example.com` saves it instead. |
| `FLAGGR_API_TOKEN` | The saved token | The token every command sends |
| `FLAGGR_EVAL_URL` | `https://api.flaggr.dev` with flaggr.dev, otherwise the API URL | Where `flaggr eval` sends evaluations |

## Development

```sh
make build   # ./flaggr
make test    # go test -race
make vuln    # govulncheck
```

`internal/flaggrv1` is generated from `proto/`: the messages and Connect-RPC clients of the parts of Flaggr's API the CLI calls (`flags.proto`; `evaluation.proto`, `EvaluationService`; `streaming.proto`, `FlagStreamService`). After changing a `.proto` file, regenerate it with [buf](https://buf.build/docs/installation):

```sh
buf generate   # or: make generate
```

`buf.gen.yaml` pins the plugins (protoc-gen-go v1.36.11 and protoc-gen-connect-go v1.20.0), so unchanged sources generate the committed code. To use protoc instead, install those two plugins and run:

```sh
protoc -I proto \
  --go_out=internal/flaggrv1 --go_opt=paths=source_relative \
  --connect-go_out=internal/flaggrv1 --connect-go_opt=paths=source_relative \
  proto/flags.proto proto/evaluation.proto proto/streaming.proto
```

The only difference from buf's output is the protoc version in each file's header.

### Third-party notices

`THIRD_PARTY_NOTICES` holds the licenses of Go and of the modules compiled into `flaggr`, and every release archive ships it. After changing dependencies, regenerate it with `make notices` (`scripts/third-party-notices.sh`): CI fails while it's out of date.

### Releases

Pushing a `v*` tag runs [GoReleaser](https://goreleaser.com) (`.github/workflows/release.yml`), which builds the archives and `checksums.txt` and publishes them as a GitHub release; it stamps the version and commit that `flaggr --version` prints. `make snapshot` (`goreleaser release --snapshot --clean`) builds the same archives in `dist/` without publishing anything.

## Security

Please report vulnerabilities privately: see [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE). The binaries also contain Go's standard library and other Go modules, under their own licenses: see [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES).
