FROM scratch

EXPOSE 8080

COPY index.html /
COPY dist/herder /

ENTRYPOINT ["/herder"]
