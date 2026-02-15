# Tomorrow: EKS Infra Work

## Goal
Adapt `kfin` for AWS/EKS infrastructure usage and cost visibility.

## Plan
1. Confirm cluster access and auth (`aws eks update-kubeconfig`, IAM/IRSA as needed).
2. Decide metrics source (`Prometheus` in-cluster vs `Amazon Managed Prometheus`).
3. Add AWS pricing model inputs (instance, EBS, optional network/NAT).
4. Add EKS-focused TUI breakdowns (namespace/team, nodegroup, spot vs on-demand).
5. Validate release/deploy workflow for target environment.

## Release Security Hardening
1. Sign release artifacts with `cosign` and publish verification instructions.
2. Generate and publish SBOMs (e.g., CycloneDX/SPDX) for each release archive.
3. Publish SLSA provenance/attestations for release builds.
4. Pin GitHub Actions to commit SHAs (not only tags like `@v4`) in release workflows.
5. Add a private-repo download section in `README.md` using authenticated `gh release download`.
6. Enforce branch protection + required checks before release publishing.
7. Add dependency and vuln scanning gates (`govulncheck`/Trivy) to block vulnerable releases.
8. Rotate and minimize repo secrets/tokens; prefer GitHub OIDC where possible.
9. Add a post-release verification checklist (hash verify, signature verify, version sanity check).
