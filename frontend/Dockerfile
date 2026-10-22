# syntax=docker/dockerfile:1
# Build context is the repo root so the Caddyfile in deploy/ can be baked in

# ---- build ----
FROM node:22 AS build
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --fund=false
COPY frontend/ ./
ARG VITE_API_URL=/api
ENV VITE_API_URL=${VITE_API_URL}
RUN npm run build

# ---- runtime ----
FROM caddy:2.11
COPY deploy/Caddyfile /etc/caddy/Caddyfile
COPY --from=build /src/dist /srv
