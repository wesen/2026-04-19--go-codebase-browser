FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app
COPY --chown=nonroot:nonroot bin/codebase-browser /app/codebase-browser

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/codebase-browser"]
CMD ["serve", "--addr", ":8080"]
