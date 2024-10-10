ARG DOCKER_PLATFORM
ARG DOCKER_ARCH

FROM --platform=${DOCKER_PLATFORM:-linux/amd64} ${DOCKER_ARCH:-amd64}/golang:1.22 AS builder
ARG CGO_ENABLED=0
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY ./supervisor ./supervisor
COPY ./cmd/supervisor ./cmd/supervisor
RUN mkdir -p /app/bin
RUN go build -o ./bin/supervisor ./cmd/supervisor

FROM --platform=${DOCKER_PLATFORM:-linux/amd64} ${DOCKER_ARCH:-amd64}/debian:bookworm-slim
ARG GOARCH
ARG VERSION_SERVER_URL
ARG SUPERVISOR_SERVER
ENV GOARCH=${GOARCH:-amd64} \
    VERSION_SERVER_URL=${VERSION_SERVER_URL:-https://version.storj.io} \
    SUPERVISOR_SERVER=${SUPERVISOR_SERVER:-unix}

RUN apt-get update
RUN apt-get install -y --no-install-recommends ca-certificates
RUN update-ca-certificates

COPY docker/ /

RUN mkdir -p /app/bin
COPY --from=builder /app/bin/supervisor /app/bin/supervisor

EXPOSE 28967
EXPOSE 14002

WORKDIR /app
ENTRYPOINT ["/entrypoint"]

ENV ADDRESS="" \
    EMAIL="" \
    WALLET="" \
    STORAGE="2.0TB" \
    SETUP="false" \
    AUTO_UPDATE="true" \
    LOG_LEVEL="" \
    BINARY_STORE_DIR="/app/config/bin" \
