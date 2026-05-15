# 🏓 DevOps Home Assignment: Ping-Pong Game Deployment

## 📋 Overview

**⏱️ Duration:** 2-3 hours  
**🎯 Objective:** Create a production-ready CI/CD pipeline that builds, containerizes, and deploys a Go application to Kubernetes.

---

## 🚀 Application

A Go HTTP server implementing a ping-pong game with these endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/ping` | GET | Returns "pong" message |
| `/pong` | GET | Returns "ping" message |
| `/health` | GET | Health check endpoint |
| `/` | GET | API documentation |

**Environment Variables:**
- `PORT` - Server port (default: 8080)
- `SECRET_FILE_PATH` - Path to secret file

**Run Modes:** `server` or `cli`

**Authentication:**
- `Authorization` header with secret token is required for `/ping` and `/pong` endpoints
- in CLI mode, the secret token is passed as a command line argument

**Note:**
- The server will think for 10 seconds before starting the server
- health check endpoint is available at `/health` and it will return 200 OK if the server is ready to serve requests
- The server will be available on the port specified in the `PORT` environment variable
- The server will read the secret token from the `SECRET_FILE_PATH` environment variable
- The secret token is passed as a command line argument in CLI mode

---

## 🎯 Mission

Take this application to production with support for both **x86** and **ARM64** architectures. Have a binary release and a container release available for developers and production.

---

## 📋 Requirements

### 🔒 Security
- [ ] No containers running as root
- [ ] All images must pass security scans
- [ ] No critical/high vulnerabilities should be released to production
- [ ] No secrets in codebase
- [ ] Proper filesystem isolation

### ☸️ Kubernetes
- [ ] Zero-downtime deployments server must be available and ready at all times
- [ ] ARM64 architecture preferred
- [ ] No direct internet access (use ingress/proxy)
- [ ] Cluster can pull from registry

### 🏗️ CI/CD
- [ ] Multi-architecture builds (x86/ARM64)
- [ ] Images stored in GitHub Container Registry
- [ ] Versioned releases with tags
- [ ] Both container and binary releases

---

## 🛠️ Environment

**Prerequisites:**
- Docker
- Minikube or Kind
- kubectl
- Go 1.24
- GitHub account

---

## 📊 Evaluation

### Technical Implementation
- Container Security
- Kubernetes Manifests and best practices
- CI/CD Pipeline container and binary releases
- Multi-Architecture builds 
- Security Scanning and release prevention for critical and high vulnerabilities

### Understanding & Explanation
- Architecture decisions
- Scaling strategy
- Cloud deployment considerations
- Security measures
- Maintaining image versions and tags and removing old ones

---

## 📝 Deliverables

- [ ] `Dockerfile`
- [ ] `k8s/` manifests
- [ ] `.github/workflows/` CI/CD pipeline
- [ ] Documentation of your approach

**Note:** Use Minikube/Kind for testing. Be prepared to explain real cloud deployment strategy.

## You will be asked to explain the following:
- The deployment strategy
- The scaling strategy
- The security measures
- The CI/CD pipeline
- The multi-architecture builds
- The versioning and tagging strategy
- Going cloud with EKS and how to deploy the application to EKS
- How to allow teams from across the world to pull the image fast using AWS solutions
- How to manage older and stale versions of the application

## Submission
- Create a fork of this repository and give access to your fork
- Once you notify that you are done no more commits!

---

**Good luck! 🚀**

---

## Implemented Solution

This repository now includes the production assets requested by the assignment:

- `Dockerfile` builds a static Go binary and runs it in a distroless nonroot image.
- `charts/echo-pong` contains the single Kubernetes workload manifest source.
- `.github/workflows/ci.yml` validates PRs with Go checks, tests, Docker build, and image scanning.
- `.github/workflows/release.yml` publishes multi-architecture GHCR images and binary GitHub releases from semver tags.
- `docs/production-readiness.md` explains deployment, scaling, security, EKS, global image distribution, and stale-version management.

Quick local flow:

```bash
go test ./...
docker build -t echo-pong:local .
helm lint ./charts/echo-pong
helm template echo-pong ./charts/echo-pong --namespace echo-pong
kubectl create namespace echo-pong
kubectl create secret generic echo-pong-secret --namespace echo-pong --from-literal=token='replace-with-a-strong-token'
helm upgrade --install echo-pong ./charts/echo-pong --namespace echo-pong --set image.repository=echo-pong --set image.tag=local --set secret.existingSecret=echo-pong-secret
```

The Helm chart is the only Kubernetes application manifest source. Namespace creation and secret creation are deployment prerequisites, because secret values should not live in the chart or repository.
