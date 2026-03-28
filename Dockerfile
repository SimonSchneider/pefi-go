FROM gcr.io/distroless/static-debian11
COPY build/app /app
VOLUME /db
EXPOSE 80
ENTRYPOINT ["/app", "-addr", ":80", "-dburl", "file:/db/pefi.sqlite?cache=shared"]
