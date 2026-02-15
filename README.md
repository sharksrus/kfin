# kfin

`kfin` is a Kubernetes cost visibility CLI with:
- terminal text analysis (`analyze`)
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

## Download Prebuilt Binaries

Grab binaries from the GitHub Releases page:

- Latest release page: https://github.com/newman-bot/kfin/releases/latest
- Linux amd64: https://github.com/newman-bot/kfin/releases/latest/download/kfin_linux_amd64.tar.gz
- Linux arm64: https://github.com/newman-bot/kfin/releases/latest/download/kfin_linux_arm64.tar.gz
- macOS arm64: https://github.com/newman-bot/kfin/releases/latest/download/kfin_darwin_arm64.tar.gz

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

Interactive dashboard:

```bash
./kfin tui
```

Export PDF report:

```bash
./kfin pdf
./kfin pdf -o kfin-report.pdf
```

## CI/CD

- Pull requests run lint/build checks via GitHub Actions.
- Published releases build and attach binaries for:
  - `linux/amd64`
  - `linux/arm64`
  - `darwin/arm64`

## License

MIT
