FROM debian:11

RUN useradd -r -u 1001 -g 0 -d /app -s /bin/false waarp

COPY ./build/waarp-gateway-docker /app

RUN chown -R :0 /app && \
    chmod -R g+rwX /app

VOLUME ["/app/etc", "/app/data"]
EXPOSE 8080/tcp
WORKDIR /app

USER 1001

ENTRYPOINT ["/app/bin/container-entrypoint"]
