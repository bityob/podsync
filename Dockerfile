# syntax=docker/dockerfile:1.7

# ---------- yt-dlp downloader ----------
# Separate stage so that changing Go source code does not invalidate the
# yt-dlp download layer, and vice versa.
FROM alpine:3.22 AS ytdl
RUN apk --no-cache add wget ca-certificates
ARG YTDLP_VERSION=latest
RUN if [ "$YTDLP_VERSION" = "latest" ]; then \
        url="https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp"; \
    else \
        url="https://github.com/yt-dlp/yt-dlp/releases/download/${YTDLP_VERSION}/yt-dlp"; \
    fi && \
    wget -O /usr/bin/yt-dlp "$url" && \
    chmod a+rx /usr/bin/yt-dlp

# ---------- Go builder ----------
FROM golang:1.25 AS builder

WORKDIR /build

# 1. Pre-download modules in their own layer. This layer is only invalidated
#    when go.mod / go.sum change, not on every source edit.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 2. Copy the rest of the sources. A cache mount keeps Go's incremental
#    build cache between image builds, so subsequent rebuilds only recompile
#    the packages whose sources actually changed.
COPY . .

ARG TAG=nightly
ARG COMMIT=""
ENV TAG=${TAG} COMMIT=${COMMIT}

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    make build

# ---------- Final runtime image ----------
# Alpine 3.22 will go EOL on 2027-05-01
FROM alpine:3.22

WORKDIR /app

# deno is required for yt-dlp (ref: https://github.com/yt-dlp/yt-dlp/issues/14404)
RUN apk --no-cache add ca-certificates python3 py3-pip ffmpeg tzdata libc6-compat deno

RUN chmod 777 /usr/local/bin
COPY --from=ytdl    /usr/bin/yt-dlp            /usr/local/bin/youtube-dl
COPY --from=builder /build/bin/podsync         /app/podsync
COPY --from=builder /build/html/index.html     /app/html/index.html

ENTRYPOINT ["/app/podsync"]
CMD ["--no-banner"]
