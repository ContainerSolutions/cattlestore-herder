FROM scratch

EXPOSE 8080

LABEL "traefik.enable=false"

COPY index.html /
COPY dist/herder /

ENTRYPOINT ["/herder"]
