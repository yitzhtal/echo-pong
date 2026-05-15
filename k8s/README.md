# Kubernetes Deployment

Create the secret outside Git before applying the manifests:

```bash
kubectl create namespace echo-pong
kubectl create secret generic echo-pong-secret \
  --namespace echo-pong \
  --from-literal=token='replace-with-a-strong-token'
```

Deploy with Kustomize:

```bash
kubectl apply -k k8s/base
```

For a local Minikube ingress test:

```bash
minikube addons enable ingress
echo "$(minikube ip) echo-pong.local" | sudo tee -a /etc/hosts
curl -H 'Authorization: Bearer replace-with-a-strong-token' http://echo-pong.local/ping
```

Deploy with Helm:

```bash
helm upgrade --install echo-pong ./charts/echo-pong \
  --namespace echo-pong \
  --create-namespace \
  --set image.repository=ghcr.io/OWNER/echo-pong \
  --set image.tag=v0.1.0 \
  --set secret.existingSecret=echo-pong-secret
```

The default manifests intentionally do not create the secret value. In production, use External Secrets Operator with AWS Secrets Manager, SSM Parameter Store, Vault, or the platform-approved secret manager.
