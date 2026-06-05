# syntax=docker/dockerfile:1
#
# Multi-stage build: build the React/Primer UI, embed it into the Go server,
# and ship a tiny distroless image.

# 1) Build the web UI -> server/internal/webui/dist (via vite outDir).
FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /src
COPY web/package.json web/package-lock.json ./web/
RUN --mount=type=cache,target=/root/.npm cd web && npm ci
RUN mkdir -p server/internal/webui
COPY web/ ./web/
RUN cd web && npm run build

# 2) Build the Go server with the embedded UI.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
WORKDIR /src
COPY server/go.mod server/go.sum ./server/
RUN --mount=type=cache,target=/go/pkg/mod cd server && go mod download
COPY server/ ./server/
COPY --from=web /src/server/internal/webui/dist ./server/internal/webui/dist
ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    cd server && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/opentdm-server ./cmd/opentdm-server

# 3) Runtime image.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/opentdm-server /opentdm-server
EXPOSE 8080
USER nonroot:nonroot
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD ["/opentdm-server", "healthcheck"]
ENTRYPOINT ["/opentdm-server"]
CMD ["serve"]
