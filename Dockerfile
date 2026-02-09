FROM golang:1.25.5 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG BUILDPLATFORM

ARG GIT_VERSION=unknown
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

RUN apt-get update && apt-get install -y gcc libc6-dev libsqlite3-dev

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -ldflags \
    "-linkmode external -extldflags '-static' \
     -X github.com/nvidia/nvsentinel/pkg/util/version.GitVersion=${GIT_VERSION} \
     -X github.com/nvidia/nvsentinel/pkg/util/version.GitCommit=${GIT_COMMIT} \
     -X github.com/nvidia/nvsentinel/pkg/util/version.BuildDate=${BUILD_DATE}" \
    -tags "sqlite_dbstat sqlite_foreign_keys libsqlite3" \
    -o device-apiserver ./cmd/device-apiserver

FROM gcr.io/distroless/base-debian12:latest AS runtime

USER 65532:5000

WORKDIR /
COPY --from=builder /workspace/device-apiserver /usr/local/bin/device-apiserver


ENTRYPOINT ["/usr/local/bin/device-apiserver"]
