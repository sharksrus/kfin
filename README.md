# Pod Cost Analyzer

Track resource waste and estimate costs for pods in your K3s/Kubernetes clusters.

## Features

- **Pod Analysis**: List all pods with their resource requests and limits
- **Cluster Status**: Quick view of nodes and pod count
- **Cost Estimation**: (Coming soon) Map resources to AWS pricing
- **Waste Detection**: (Coming soon) Identify over-provisioned pods
- **Rightsizing Suggestions**: (Coming soon) Recommend resource adjustments

## Quick Start

### Build

```bash
go build -o pod-cost-analyzer
```

### Usage

Check cluster status:
```bash
./pod-cost-analyzer status
```

Analyze pod costs:
```bash
./pod-cost-analyzer analyze
```

## Architecture

```
main.go
├── cmd/
│   ├── analyze.go      # Pod enumeration & resource analysis
│   └── status.go       # Cluster health check
├── pkg/
│   ├── metrics/        # Prometheus metric collection (TODO)
│   ├── pricing/        # Cost calculation engine (TODO)
│   └── suggester/      # Rightsizing logic (TODO)
└── go.mod
```

## Development Roadmap

### Phase 1: MVP (This Week)
- [x] K3s cluster connectivity
- [x] Pod enumeration & resource parsing
- [ ] Prometheus metrics scraping
- [ ] Cost calculation (dummy pricing)
- [ ] CSV/JSON output

### Phase 2: AWS Integration (Next Week)
- [ ] AWS pricing API integration
- [ ] Real cost mapping to EC2/Fargate
- [ ] Cost per namespace/team
- [ ] Web dashboard (optional)

## Cluster Setup

Assumes kubeconfig at `~/.kube/config`. Set `KUBECONFIG` env var if different.

```bash
export KUBECONFIG=/path/to/kubeconfig
./pod-cost-analyzer status
```

## Next Steps

1. **Tomorrow**: Wire up kubeconfig access to your K3s cluster
2. **Week 1**: Add Prometheus scraper for actual resource usage
3. **Week 2**: Implement cost calculation + AWS pricing integration

## License

MIT
