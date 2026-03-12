# Project: i-love-shopping

## Tech Stack

- Backend: Go (Chi router, pgx/v5, golang-migrate)
- Database: PostgreSQL (via Docker)
- Auth: JWT (access + refresh tokens), bcrypt, UUID
- Architecture: Clean layered (handler → service → repository)
- Containerization: Docker + docker-compose

## Architecture Rules

- Handler layer: HTTP only, no business logic
- Service layer: Business logic only, no SQL
- Repository layer: DB only, returns domain errors
- Never skip layers

## Code Style

- Explicit error handling, no silent failures
- Domain errors mapped at the repository layer
- Interfaces for all dependencies (for testability)
- Follow Go standard project layout conventions

## What NOT to do

- No ORMs — raw SQL with pgx
- No shortcuts on JWT — follow spec exactly (memory storage, rotation, revocation)
- No untested code merged to main

## Current State

- [update as you build]
