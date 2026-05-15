# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25.10

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
	-trimpath \
	-ldflags="-s -w -buildid=" \
	-o /out/echo-pong .

FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="echo-pong" \
	org.opencontainers.image.description="Production-ready ping-pong Go service"

WORKDIR /
USER 65532:65532

COPY --from=build --chown=65532:65532 /out/echo-pong /echo-pong

EXPOSE 8080
ENV PORT=8080 \
	SECRET_FILE_PATH=/var/run/secrets/echo-pong/token

ENTRYPOINT ["/echo-pong"]
