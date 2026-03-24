# --- Build stage ---
FROM golang:alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

ENV GOPRIVATE=gitea.kood.tech/*
ENV GONOSUMCHECK=gitea.kood.tech/*

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

# --- Run stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /api /app/api
COPY migrations /app/migrations
COPY static /app/static

EXPOSE 8080

CMD ["/app/api"]
