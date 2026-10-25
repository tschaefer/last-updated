FROM --platform=$BUILDPLATFORM golang:1.27.3-trixie AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc-aarch64-linux-gnu libc6-dev-arm64-cross \
    gcc-x86-64-linux-gnu libc6-dev-amd64-cross \
    make ca-certificates && apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /build

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN if [ "${TARGETARCH}" = "arm64" ]; then \
        export CC=aarch64-linux-gnu-gcc; \
    elif [ "${TARGETARCH}" = "amd64" ]; then \
        export CC=x86_64-linux-gnu-gcc; \
    else \
        echo "Unsupported architecture: ${TARGETARCH}" && exit 1; \
    fi && \
    make dist GOOS=${TARGETOS} GOARCH=${TARGETARCH}

FROM debian:trixie-20261020-slim

ARG TARGETOS
ARG TARGETARCH

COPY --from=builder /build/bin/last-updated-${TARGETOS}-${TARGETARCH} /bin/last-updated
EXPOSE 8080
EXPOSE 8443
EXPOSE 9090

ENTRYPOINT ["/bin/last-updated"]
