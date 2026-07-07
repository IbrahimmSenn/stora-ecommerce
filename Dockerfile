# --- Frontend build stage ---
FROM node:22-alpine AS web-builder

WORKDIR /web

COPY web/package.json web/package-lock.json ./

RUN npm ci

COPY web/ ./

RUN npm run build

# --- Backend build stage ---
FROM golang:alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

ENV GOPRIVATE=gitea.kood.tech/*
ENV GONOSUMCHECK=gitea.kood.tech/*

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

# --- Backend test stage (not part of the runtime image) ---
# Built explicitly by CI with `docker build --target test .` so the suite runs
# in the same environment the binary is compiled in.
FROM builder AS test

RUN go vet ./... && go test ./... -count=1

# --- Run stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /api /app/api
COPY migrations /app/migrations
COPY --from=web-builder /web/dist /app/web/dist

# Run as an unprivileged user. The runtime-writable dirs (uploads, self-signed
# certs) are created here and owned by appuser so the mounted named volumes
# inherit that ownership on first creation.
RUN addgroup -S app && adduser -S -G app appuser \
	&& mkdir -p /app/uploads /app/certs \
	&& chown -R appuser:app /app

USER appuser

EXPOSE 8080

CMD ["/app/api"]
