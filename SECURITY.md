# Security Policy

## Reporting a Vulnerability

Please report suspected vulnerabilities privately.

- Preferred: GitHub Security Advisories for this repository.
- Alternate: contact repository maintainers directly.

Do not open a public issue for undisclosed vulnerabilities.

## Release Artifact Integrity

Releases publish the following assets per platform:

- `kfin_<os>_<arch>.tar.gz`
- `kfin_<os>_<arch>.tar.gz.sha256`
- `kfin_<os>_<arch>.tar.gz.sig`
- `kfin_<os>_<arch>.tar.gz.sigstore.json`

Verify artifacts before use:

```bash
shasum -a 256 -c kfin_darwin_arm64.tar.gz.sha256

cosign verify-blob \
  --signature kfin_darwin_arm64.tar.gz.sig \
  --bundle kfin_darwin_arm64.tar.gz.sigstore.json \
  --certificate-identity "https://github.com/sharksrus/kfin/.github/workflows/release.yml@refs/tags/v0.0.1" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  kfin_darwin_arm64.tar.gz
```

## Build and Workflow Security Controls

- GitHub Actions in CI/release are pinned to immutable commit SHAs.
- Release workflow performs `govulncheck ./...` before build.
- Release artifacts are keylessly signed using Sigstore Cosign via GitHub OIDC.
