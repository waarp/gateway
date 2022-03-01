FROM debian:11
# TODO add maintainer labels
# TODO use a non root user
COPY ./build/waarp-gateway-docker /app

VOLUME ["/app/etc", "/app/data"]
EXPOSE 8080/tcp
WORKDIR /app

ENTRYPOINT ["/app/bin/container-entrypoint"]
