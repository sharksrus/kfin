# Tomorrow: EKS Infra Work

## Goal
Adapt `kfin` for AWS/EKS infrastructure usage and cost visibility.

## Plan
1. Confirm cluster access and auth (`aws eks update-kubeconfig`, IAM/IRSA as needed).
2. Decide metrics source (`Prometheus` in-cluster vs `Amazon Managed Prometheus`).
3. Add AWS pricing model inputs (instance, EBS, optional network/NAT).
4. Add EKS-focused TUI breakdowns (namespace/team, nodegroup, spot vs on-demand).
5. Validate release/deploy workflow for target environment.
