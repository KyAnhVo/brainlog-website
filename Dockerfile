# syntax=docker/dockerfile:1
# One image: the Go API with both frontends embedded.
#   /        customer site      /admin/  admin panel      /api/...  API
# Configure at runtime with env vars (DB_ADDR, ADDR, COOKIE_SECURE, ...); no .env file is baked in.

FROM oven/bun:1.4.4 AS web
WORKDIR /src
COPY customer-facing-client/package.json customer-facing-client/bun.lock customer-facing-client/
COPY admin-panel-client/package.json admin-panel-client/bun.lock admin-panel-client/
RUN cd customer-facing-client && bun install --frozen-lockfile \
 && cd ../admin-panel-client && bun install --frozen-lockfile
COPY customer-facing-client customer-facing-client
COPY admin-panel-client admin-panel-client
RUN cd customer-facing-client && bun run build \
 && cd ../admin-panel-client && bun run build \
 && rm -f */dist/mockServiceWorker.js

FROM golang:1.27 AS server
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server ./
COPY --from=web /src/customer-facing-client/dist internal/web/dist/customer
COPY --from=web /src/admin-panel-client/dist internal/web/dist/admin
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/brainlog-server .

# distroless/static includes CA certificates (needed for TLS to the database) and runs as nonroot.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=server /out/brainlog-server /brainlog-server
ENV ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/brainlog-server"]
