# Stage 1: Build frontend
FROM node:22-bookworm-slim AS frontend-builder

# pnpm 11 ships a native binary that needs libatomic (absent from the slim image)
RUN apt-get update && apt-get install -y --no-install-recommends libatomic1 && rm -rf /var/lib/apt/lists/*
RUN npm install -g pnpm@11.5.2

# pnpm 11 refuses to purge node_modules without a TTY unless CI is set
ENV CI=true

WORKDIR /app
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml frontend/.npmrc ./frontend/
RUN pnpm install --dir frontend --frozen-lockfile

COPY frontend/ ./frontend/
RUN pnpm --dir frontend build

# Stage 2: Build Go backend with embedded frontend
FROM golang:1.26-bookworm AS backend-builder

WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download

ARG APP_VERSION=0.0.0

COPY backend/ ./
COPY --from=frontend-builder /app/dist ./internal/assets/web/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X 'github.com/Jellystics/Jellystics/internal/handler.currentAppVersion=${APP_VERSION}'" \
    -o /jellystics ./cmd/server

# Stage 3: Minimal runtime image
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -yqq --no-install-recommends ca-certificates wget && \
    apt-get autoremove && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=backend-builder /jellystics .
COPY --chmod=755 entry.sh /entry.sh

HEALTHCHECK --interval=30s \
            --timeout=5s \
            --start-period=10s \
            --retries=3 \
            CMD [ "/usr/bin/wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:3000/auth/isConfigured" ]

EXPOSE 3000

CMD ["/entry.sh"]
