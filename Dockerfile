FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app
COPY --chown=nonroot:nonroot bin/codebase-browser /app/codebase-browser
COPY --chown=nonroot:nonroot bin/history.db /app/history.db

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/codebase-browser"]
CMD ["serve", "--addr", ":8080", "--history-db", "/app/history.db", "--repo-root", "/app"]
