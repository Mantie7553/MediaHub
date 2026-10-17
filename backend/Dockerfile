# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mediahub ./cmd/api

# ---- runtime ----
FROM debian:trixie-slim

ARG YTDLP_VERSION=2026.08.19
ARG MANGAL_VERSION=4.0.6

# ffmpeg/ffprobe for streaming (Debian's build has libx264 and VAAPI), mesa for AMD/Intel VAAPI
RUN apt-get update \
 && apt-get install -y --no-install-recommends ffmpeg mesa-va-drivers ca-certificates curl \
 && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL -o /usr/local/bin/yt-dlp \
      "https://github.com/yt-dlp/yt-dlp/releases/download/${YTDLP_VERSION}/yt-dlp_linux" \
 && chmod 0755 /usr/local/bin/yt-dlp \
 && curl -fsSL "https://github.com/metafates/mangal/releases/download/v${MANGAL_VERSION}/mangal_${MANGAL_VERSION}_Linux_x86_64.tar.gz" \
      | tar -xz -C /usr/local/bin mangal \
 && chmod 0755 /usr/local/bin/mangal

COPY --from=build /out/mediahub /usr/local/bin/mediahub
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod 0755 /usr/local/bin/docker-entrypoint.sh

ENV YTDLP_PATH=/usr/local/bin/yt-dlp \
    MANGAL_PATH=/usr/local/bin/mangal \
    HOME=/data
WORKDIR /data

EXPOSE 9090
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD curl -fsS http://localhost:9090/health || exit 1

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["mediahub"]
