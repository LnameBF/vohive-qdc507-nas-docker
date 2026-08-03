FROM docker.io/library/node:20-alpine AS frontend-builder

WORKDIR /src/vohive/web
COPY third_party/vohive/web/package*.json ./
RUN npm ci
COPY third_party/vohive/web/ ./
RUN ulimit -n 65536 && npm run build

FROM docker.io/library/golang:1.26-alpine AS backend-builder

WORKDIR /src
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOTOOLCHAIN=auto \
    GOPROXY=${GOPROXY}
RUN apk add --no-cache git

# Keep the upstream source layout intact: VoHive's go.mod replaces
# ../vowifi-go with the sibling source tree.
COPY third_party/vowifi-go ./vowifi-go
COPY third_party/swu-go ./swu-go
COPY third_party/vohive ./vohive
COPY --from=frontend-builder /src/vohive/web/dist /src/vohive/internal/web/dist

WORKDIR /src/vohive
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=mod \
      -trimpath \
      -buildvcs=false \
      -tags 'with_utls nomsgpack' \
      -ldflags "-s -w -X 'github.com/iniwex5/vohive/internal/global.Version=v0.2.1-qdc507'" \
      -o /out/vohive \
      ./cmd/vohive

FROM docker.io/library/ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        iproute2 \
        kmod \
        libqmi-proxy \
        libqmi-utils \
        usbutils \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/vohive

COPY --from=backend-builder /out/vohive /opt/vohive/bin/vohive
COPY config.example.yaml /opt/vohive/config/config.yaml
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
COPY scripts/verify-qmi.sh /usr/local/bin/verify-qmi

RUN chmod 0755 /opt/vohive/bin/vohive /usr/local/bin/docker-entrypoint.sh /usr/local/bin/verify-qmi \
    && mkdir -p /opt/vohive/data /opt/vohive/logs \
    && test -x /usr/libexec/qmi-proxy

EXPOSE 7575
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
