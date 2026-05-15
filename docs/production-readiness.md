# Production Readiness Notes

## Architecture Decisions

The service is a small stateless Go API, so the production shape is deliberately simple:

- Static Go binary built for `linux/amd64` and `linux/arm64`.
- Distroless nonroot container runtime.
- Kubernetes `Deployment` behind a `ClusterIP` `Service` and `Ingress`.
- Both a Helm chart and Kustomize-ready `k8s/base` manifests are included as application manifest sources.
- Token loaded from a mounted Kubernetes `Secret`; no secret value is committed.
- Readiness, liveness, graceful shutdown, resource limits, HPA, PDB, and NetworkPolicy are all part of the default deployment.

## Deployment Strategy

The Kubernetes deployment uses a rolling update with `maxUnavailable: 0` and `maxSurge: 1`. This keeps old pods serving while a new pod starts, passes startup, and becomes ready.

The application exposes:

- `/health` for liveness.
- `/ready` for readiness.

On `SIGTERM`, the app marks itself unready, waits briefly for endpoint propagation, and then gracefully shuts down the HTTP server. This reduces dropped requests during rolling deployments, node drains, and cluster autoscaler activity.

Production releases should deploy immutable image references. Tags are friendly for humans, but the safest production deployment is pinning the image digest emitted by the release pipeline.

## Scaling Strategy

The default deployment starts with two replicas, a PDB with `minAvailable: 1`, and an HPA scaling from 2 to 6 pods on CPU and memory utilization.

For cloud production:

- Use at least two Availability Zones.
- Keep topology spread constraints enabled.
- Prefer ARM64/Graviton nodes for cost efficiency, with amd64 available as fallback.
- Run Cluster Autoscaler or Karpenter so HPA demand can add nodes.
- Review request/limit values after observing real latency, CPU, and memory profiles.

## Security Measures

Container security:

- Runtime image is distroless and runs as UID/GID `65532`.
- No shell or package manager is present in the final image.
- Root filesystem is read-only in Kubernetes.
- Linux capabilities are dropped.
- Privilege escalation is disabled.
- The default seccomp profile is required.

- Kubernetes security:
- The deployment procedure labels the namespace to enforce the `restricted` Pod Security Standard.
- Service account token automounting is disabled.
- Secret is mounted read-only from an existing Kubernetes secret.
- NetworkPolicy denies ingress and egress by default, then allows only ingress-controller traffic to the app.
- Service is `ClusterIP`; users reach the app through Ingress/proxy, not direct pod or node exposure.

CI/CD security:

- Pull requests run Go formatting, vetting, race-enabled tests, image build, and vulnerability scan.
- Release image scanning fails on `high` and `critical` findings before pushing the multi-arch production image.
- Dependabot is enabled for GitHub Actions, Go modules, and Docker base images.
- For a stricter production setup, pin third-party GitHub Actions to full commit SHAs and rotate them through Dependabot-reviewed updates.

## CI/CD Pipeline

`ci.yml` runs on pull requests and pushes to `main`:

- `gofmt`
- `go vet`
- `go test -race -cover`
- Helm lint and template rendering
- local Docker build
- vulnerability scan with high/critical release prevention

`release.yml` runs only on semantic version tags such as `v1.2.3`:

- builds binary archives for Linux, macOS, and Windows on amd64 and arm64
- creates checksums
- scans a local Linux image
- pushes a multi-architecture image to GHCR
- publishes a GitHub Release with the binary artifacts

## Multi-Architecture Builds

Docker Buildx builds `linux/amd64` and `linux/arm64` images from the same Dockerfile. The Dockerfile uses BuildKit platform arguments so the Go binary is compiled for the target platform inside the container build.

Binary releases are built with explicit `GOOS` and `GOARCH` values. This gives developers local CLI binaries while production gets a multi-arch container.

## Versioning And Tagging

Release source of truth is a Git tag: `vMAJOR.MINOR.PATCH`.

The container release publishes:

- full semver tag, for example `v1.2.3`
- minor stream tag, for example `1.2`
- commit tag, for example `sha-<git-sha>`

Recommended deployment policy:

- Development can use semver tags.
- Staging should use the exact release tag.
- Production should pin the image digest after the release scan passes.

## EKS Deployment Strategy

For EKS, use the Helm chart as the install unit:

```bash
helm upgrade --install echo-pong ./charts/echo-pong \
  --namespace echo-pong \
  --create-namespace \
  --set image.repository=<account>.dkr.ecr.<region>.amazonaws.com/echo-pong \
  --set image.tag=v1.2.3 \
  --set ingress.className=alb
```

Recommended AWS components:

- EKS managed node groups or Karpenter with ARM64 Graviton node pools.
- AWS Load Balancer Controller for ALB Ingress.
- External Secrets Operator with AWS Secrets Manager or SSM Parameter Store.
- IRSA for controllers that need AWS permissions.
- ECR as the regional production registry mirror.
- VPC endpoints for ECR, S3, CloudWatch, and Secrets Manager in private clusters.
- CloudWatch Container Insights, Prometheus, or another metrics stack for HPA and operational visibility.

## Fast Global Image Pulls

GHCR satisfies the assignment requirement. In AWS production, mirror released images into Amazon ECR and enable cross-region replication to regions close to the teams and clusters that pull the image.

For global teams:

- Push the release image once, then replicate it to regional ECR registries.
- Configure each EKS cluster to pull from its local region.
- Use ECR lifecycle policies consistently across replicated repositories.
- Publish binary artifacts to S3 behind CloudFront if developers download them frequently.

## Managing Older And Stale Versions

Use a retention policy that separates human convenience from production safety:

- Keep all images currently deployed in any environment.
- Keep the latest N semver releases, for example 20 production releases.
- Keep all releases under active support.
- Delete untagged images older than a short retention window.
- Keep SBOMs, checksums, and provenance for every retained release.

On ECR, use lifecycle policies. On GHCR, use a scheduled cleanup workflow or GitHub API automation that never deletes a digest referenced by a live deployment.
