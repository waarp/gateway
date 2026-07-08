FROM debian:11

ARG VERSION=latest
LABEL org.opencontainers.image.vendor="Waarp" \
      org.opencontainers.image.title="Waarp Gateway" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.description="Waarp Gateway, the open source multi-protocol MFT gateway" \
      org.opencontainers.image.url="https://waarp.org" \
      org.opencontainers.image.source="https://code.waarp.fr/apps/gateway/gateway" \
      org.opencontainers.image.documentation="https://doc.waarp.org/waarp-gateway/${VERSION}/fr/reference/container.html" \
      org.opencontainers.image.authors="Bruno Carlin <bruno.carlin@waarp.org>" \
      org.opencontainers.image.licenses="GPL-3.0-only"

RUN useradd -r -u 1001 -g 0 -d /app -s /bin/false waarp

COPY ./build/waarp-gateway-docker /app

RUN chown -R :0 /app && \
    chmod -R g+rwX /app

VOLUME ["/app/etc", "/app/data"]
EXPOSE 8080/tcp
WORKDIR /app

USER 1001

ENTRYPOINT ["/app/bin/container-entrypoint"]
