FROM ubuntu:24.04

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

# The upstream VoHive binary is intentionally not distributed by this project.
# Before building, place a legally obtained Linux amd64 executable at
# vendor/vohive.
COPY vendor/vohive /opt/vohive/bin/vohive
COPY config.yaml /opt/vohive/config/config.yaml
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
COPY scripts/verify-qmi.sh /usr/local/bin/verify-qmi

RUN chmod 0755 /opt/vohive/bin/vohive /usr/local/bin/docker-entrypoint.sh /usr/local/bin/verify-qmi \
    && mkdir -p /opt/vohive/data /opt/vohive/logs \
    && test -x /usr/libexec/qmi-proxy

EXPOSE 7575
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
