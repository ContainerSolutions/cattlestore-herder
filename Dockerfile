FROM scratch

EXPOSE 8080

COPY dist/herder /

ENTRYPOINT ["/herder"]
