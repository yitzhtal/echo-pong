# Echo Pong Helm Chart

This chart is the single Kubernetes deployment interface for the project.

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
  --set image.tag=v0.1.0 \
  --set secret.existingSecret=echo-pong-secret
```

For a local Minikube ingress test:

```bash
minikube addons enable ingress
echo "$(minikube ip) echo-pong.local" | sudo tee -a /etc/hosts
curl -H 'Authorization: Bearer replace-with-a-strong-token' http://echo-pong.local/ping
```

The chart intentionally expects an existing secret by default. In production, use External Secrets Operator with AWS Secrets Manager, SSM Parameter Store, Vault, or the platform-approved secret manager.
