# kfin

`kfin` is a Kubernetes cost visibility CLI with:
- terminal text analysis (`analyze`)
- historical usage analysis (`history`)
- interactive TUI dashboard (`tui`)
- PDF report export (`pdf`)

## Prerequisites

- Go (for local builds)
- Access to a Kubernetes cluster
- Valid kubeconfig (default `~/.kube/config`, or `KUBECONFIG`)

## Build Locally

```bash
go build -o kfin
```

Embed version metadata in local builds:

```bash
go build -ldflags "-X main.version=v0.1.0 -X main.buildNumber=123" -o kfin
./kfin --version
```

## Development Hooks

Install repo-managed Git hooks (one-time per clone):

```bash
make hooks
```

This enables a `pre-commit` hook that:
- runs `gofmt -w` on staged `.go` files
- re-stages those formatted files before commit

## Download Prebuilt Binaries

Grab binaries from the GitHub Releases page:

- Latest release page: https://github.com/sharksrus/kfin/releases/latest
- Linux amd64: https://github.com/sharksrus/kfin/releases/latest/download/kfin_linux_amd64.tar.gz
- Linux arm64: https://github.com/sharksrus/kfin/releases/latest/download/kfin_linux_arm64.tar.gz
- macOS arm64: https://github.com/sharksrus/kfin/releases/latest/download/kfin_darwin_arm64.tar.gz

Note: binaries are published as **Release assets** (not GitHub Packages). If a direct link returns 404, check the release page first to confirm assets were attached.

Extract and run:

```bash
tar -xzf kfin_linux_amd64.tar.gz
chmod +x kfin
./kfin status
```

Each release also includes `.sha256` checksum files.

## Usage

Check connectivity:

```bash
./kfin status
```

Text analysis report:

```bash
./kfin analyze
```

Historical usage summary:

```bash
./kfin history
./kfin history --hours 168 --step 15m
./kfin history --hours 24 --step 1m --debug
```

Interactive dashboard:

```bash
./kfin tui
```

Export PDF report:

```bash
./kfin pdf
./kfin pdf -o kfin-report.pdf
```

## Shell Completion

`kfin` includes a built-in `completion` command.

Generate completion scripts:

```bash
./kfin completion zsh
./kfin completion bash
./kfin completion fish
./kfin completion powershell
```

Install examples:

```bash
# zsh
mkdir -p "${HOME}/.zfunc"
./kfin completion zsh > "${HOME}/.zfunc/_kfin"

# bash (linux)
./kfin completion bash | sudo tee /etc/bash_completion.d/kfin > /dev/null
```

## Configuration

`kfin` reads `config.yaml` from the current working directory.

For historical usage queries, configure:

```yaml
stats:
  base_url: "http://stats.kramerica.ai"
  query_timeout_seconds: 15
  default_lookback_hours: 24
```

## CI/CD

- Pull requests run lint/build checks via GitHub Actions.
- Published releases build and attach binaries for:
  - `linux/amd64`
  - `linux/arm64`
  - `darwin/arm64`

## License

MIT
