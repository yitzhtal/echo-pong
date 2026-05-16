# Echo Pong Helm Chart

This chart is the single Kubernetes deployment interface for the project.

## Kubernetes Compatibility

The chart supports the latest three upstream-supported Kubernetes minor releases. As of May 15, 2026, this is Kubernetes 1.36, 1.35, and 1.34. CI installs and tests the chart on all three minor lines with k3d on Linux amd64 and Linux arm64 nodes.

Create the runtime secret outside Git:

```bash
kubectl create namespace echo-pong
kubectl label namespace echo-pong \
  pod-security.kubernetes.io/enforce=restricted \
  pod-security.kubernetes.io/enforce-version=latest \
  pod-security.kubernetes.io/audit=restricted \
  pod-security.kubernetes.io/audit-version=latest \
  pod-security.kubernetes.io/warn=restricted \
  pod-security.kubernetes.io/warn-version=latest \
  --overwrite
kubectl create secret generic echo-pong-secret \
  --namespace echo-pong \
  --from-literal=token='replace-with-a-strong-token'
```

Install or upgrade with Helm:

```bash
helm upgrade --install echo-pong ./charts/echo-pong \
  --namespace echo-pong \
  --set image.repository=ghcr.io/OWNER/echo-pong \
  --set image.tag=1.0.0-beta.1 \
  --set secret.existingSecret=echo-pong-secret
```

The chart defaults to AWS ALB Ingress with `ingress.provider: aws`, `ingressClassName: alb`, and AWS Load Balancer Controller annotations. For Azure, GCP, or a local ingress controller, switch to a custom provider in a small values file instead of editing templates:

```bash
cat > provider-values.yaml <<'EOF'
ingress:
  provider: custom
  className: <provider-ingress-class>
  annotations: {}
networkPolicy:
  ingressCIDRBlocks:
    - <load-balancer-or-vpc-cidr>
EOF

helm upgrade --install echo-pong ./charts/echo-pong \
  --namespace echo-pong \
  -f provider-values.yaml
```

For local testing without a cloud ingress controller, use `kubectl port-forward service/echo-pong`.

The chart intentionally expects an existing secret by default. In production, use External Secrets Operator with AWS Secrets Manager, SSM Parameter Store, Vault, or the platform-approved secret manager.
