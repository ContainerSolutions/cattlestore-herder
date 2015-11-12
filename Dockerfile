FROM alpine

EXPOSE 8080

LABEL "traefik.enable=false"

COPY index.html /
COPY dist/herder /
COPY assets /assets

ENTRYPOINT ["/herder"]
