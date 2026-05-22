# metis-cu reference container — one-shot bring-up of the full
# desktop-automation stack so users don't have to manually install
# Xvfb + fluxbox + xterm + chrome + tesseract + the metis-cu MCP
# server on a bare Linux box.
#
# Stages:
#   1. builder — compile metis-cu (CGO required, links against X11)
#   2. runtime — slim debian + Xvfb + fluxbox + chrome + tesseract +
#      a tiny supervisord-free entrypoint that brings up the GUI
#      stack on DISPLAY=:99 and then exec's metis-cu via stdio.
#
# Pull: docker pull ghcr.io/ricardo-m-l/metis-cu:latest
# Run interactively (smoke test): docker run --rm -it ghcr.io/ricardo-m-l/metis-cu:latest
# As an MCP server (typical use): wrap in your MCP client's stdio
#   config, e.g.
#     [[servers]]
#       name = "computer-use"
#       command = "docker"
#       args = ["run","--rm","-i","ghcr.io/ricardo-m-l/metis-cu:latest"]
#
# Memory: Xvfb + fluxbox + Chrome takes ~600MB; metis-cu itself is
# ~50MB. Plan for 1GB of RAM if you also want to load real web pages.

# ────────────── Stage 1: builder ──────────────
# 1.23 is the most recent tag the Tencent docker mirror reliably
# resolves (2026-05-22 attempt at 1.26-bookworm returned 500). The
# binary that ships needs Go 1.21+ only; using 1.23 is fine.
FROM golang:1.23-bookworm AS builder
WORKDIR /src

# Use the Tencent debian mirror — 2026-05-22 cn-network build saw
# deb.debian.org pulls hit 30+ min on basic packages. With the
# Tencent mirror chrome + apt deps land in <3 min total.
# Comment out when building outside CN.
RUN sed -i 's|deb.debian.org|mirrors.cloud.tencent.com|g; s|security.debian.org|mirrors.cloud.tencent.com|g' /etc/apt/sources.list.d/debian.sources

# Native deps for CGO: libxtst (XTEST), libx11 (xdo bindings),
# libxxf86vm (screenshot). Same set the README lists for local
# builds.
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      libxtst-dev libx11-dev libxxf86vm-dev gcc && \
    rm -rf /var/lib/apt/lists/*

# Cache module downloads separately from source for fast rebuilds.
# GOPROXY override: default proxy.golang.org is blocked from CN
# networks (tencent-cloud Beijing etc — observed 2026-05-22 build
# fail). goproxy.cn is the most reliable mirror; falls back to
# `direct` for any module the mirror doesn't have.
#
# GOTOOLCHAIN=auto lets `go` auto-fetch a newer toolchain (1.26+)
# when go.mod requires it — the bookworm base image only has
# 1.23 pre-installed. Without auto, the build hits
# "go.mod requires go >= 1.26.1 (running go 1.23.12)".
ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct \
    GOTOOLCHAIN=auto
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -ldflags "-s -w" -o /out/metis-cu ./.

# ────────────── Stage 2: runtime ──────────────
FROM debian:bookworm-slim
LABEL org.opencontainers.image.source="https://github.com/Ricardo-M-L/metis-cu"
LABEL org.opencontainers.image.description="metis-cu — computer-use MCP server (Linux desktop stack pre-installed)"
LABEL org.opencontainers.image.licenses="MIT"

ENV DEBIAN_FRONTEND=noninteractive \
    DISPLAY=:99 \
    CHROME_USER_DATA=/tmp/chrome-data \
    SCREEN_RES=1280x800x24

# X server + WM + minimal terminal + Chrome (apt repo) + tesseract.
# We avoid Recommends to stay slim. Chrome installs from Google's
# stable apt source so we get current security patches.
# Tencent debian mirror for runtime stage too — saves ~25 min of
# apt downloads when building from CN networks.
RUN sed -i 's|deb.debian.org|mirrors.cloud.tencent.com|g; s|security.debian.org|mirrors.cloud.tencent.com|g' /etc/apt/sources.list.d/debian.sources

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates curl gnupg \
      xvfb fluxbox xterm xdotool \
      libxtst6 libxxf86vm1 \
      tesseract-ocr tesseract-ocr-eng \
      fonts-dejavu-core \
      procps \
    && curl -fsSL https://dl.google.com/linux/linux_signing_key.pub | gpg --dearmor -o /usr/share/keyrings/google-chrome.gpg \
    && echo "deb [arch=amd64 signed-by=/usr/share/keyrings/google-chrome.gpg] https://dl.google.com/linux/chrome/deb/ stable main" > /etc/apt/sources.list.d/google-chrome.list \
    && apt-get update && apt-get install -y --no-install-recommends google-chrome-stable \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/metis-cu /usr/local/bin/metis-cu

# Entrypoint: starts Xvfb + fluxbox + (optional) Chrome with
# --remote-debugging-port=9222, then exec's metis-cu over stdio.
# Designed for `docker run -i` (MCP clients pipe JSON-RPC via stdin).
RUN cat > /usr/local/bin/start-cu.sh <<'EOF'
#!/bin/bash
set -e

# 1. X virtual framebuffer with RANDR extension (kbinani/screenshot
#    needs RANDR to enumerate display bounds; without it, every
#    screenshot returns a 0x0 image).
Xvfb :99 -screen 0 ${SCREEN_RES} -ac +extension RANDR \
  > /tmp/xvfb.log 2>&1 &
XVFB_PID=$!

# 2. Fluxbox WM — fail-closed app gates in metis-cu's tier system
#    require a WM that publishes _NET_ACTIVE_WINDOW, so xdotool can
#    identify the front-most app. Without a WM, every mouse/keyboard
#    call rejects with "frontmost app lookup failed".
sleep 1
DISPLAY=:99 fluxbox > /tmp/fluxbox.log 2>&1 &
FLUXBOX_PID=$!

# 3. Optional: pre-launch Chrome on a debug port so browser_* DOM
#    tools work out of the box. Enable by passing
#    `--env START_CHROME=1` (off by default — many use-cases want
#    an empty desktop).
sleep 1
if [ "${START_CHROME:-0}" = "1" ]; then
  mkdir -p "$CHROME_USER_DATA"
  DISPLAY=:99 google-chrome \
    --no-sandbox --disable-gpu --disable-dev-shm-usage \
    --no-first-run --no-default-browser-check \
    --remote-debugging-port=9222 \
    --window-size=1280,800 \
    --user-data-dir="$CHROME_USER_DATA" \
    "${CHROME_START_URL:-about:blank}" \
    > /tmp/chrome.log 2>&1 &
fi

# Reap subprocesses on signal so the container shuts down cleanly
# when MCP client disconnects (docker stop / EOF on stdin).
trap "kill $XVFB_PID $FLUXBOX_PID 2>/dev/null; pkill chrome 2>/dev/null; exit 0" SIGTERM SIGINT EXIT

# 4. Hand stdio to metis-cu. The MCP client (parent process) pipes
#    JSON-RPC frames here.
exec /usr/local/bin/metis-cu
EOF
RUN chmod +x /usr/local/bin/start-cu.sh

# Expose the Chrome remote-debugging port (only relevant when
# START_CHROME=1 + the MCP client wants to attach external CDP
# clients; metis-cu itself reaches the port via 127.0.0.1).
EXPOSE 9222

ENTRYPOINT ["/usr/local/bin/start-cu.sh"]
