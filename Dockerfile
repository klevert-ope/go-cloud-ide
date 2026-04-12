# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /app/data

WORKDIR /app

# The API commonly runs with /var/run/docker.sock mounted, so root is the most
# compatible default unless the socket permissions are managed externally.
COPY --from=builder /out/server /app/server
COPY templates /app/templates

EXPOSE 8090
VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -T 2 -O /dev/null http://127.0.0.1:8090/ || exit 1

ENTRYPOINT ["/app/server"]
