FROM golang:1.25.5-alpine AS builder

RUN apk add --no-cache \
    gcc \
    musl-dev \
    sqlite-dev \
    sqlite-static \
    binutils

WORKDIR /workspace

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG BUILDPLATFORM
ARG GIT_VERSION=unknown
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -mod=readonly -trimpath \
    -ldflags "-w -s -linkmode external -extldflags '-static' \
     -X github.com/nvidia/nvsentinel/pkg/util/version.GitVersion=${GIT_VERSION} \
     -X github.com/nvidia/nvsentinel/pkg/util/version.GitCommit=${GIT_COMMIT} \
     -X github.com/nvidia/nvsentinel/pkg/util/version.BuildDate=${BUILD_DATE}" \
    -tags "sqlite_dbstat sqlite_foreign_keys libsqlite3 netgo osusergo" \
    -o device-apiserver ./cmd/device-apiserver

RUN [ -z "$(scanelf --needed --noheader device-apiserver | awk '{print $2}')" ] || \
    (echo "Static link verification failed: binary has dynamic dependencies" && exit 1)

FROM gcr.io/distroless/static-debian12:latest AS runtime

USER 65532:5000

WORKDIR /
COPY --from=builder /workspace/device-apiserver /usr/local/bin/device-apiserver


ENTRYPOINT ["/usr/local/bin/device-apiserver"]
