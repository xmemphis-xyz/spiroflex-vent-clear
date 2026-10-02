FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /ventclear ./cmd/ventclear

FROM ghcr.io/home-assistant/base:3.24

ARG BUILD_VERSION
ARG BUILD_ARCH

LABEL \
  io.hass.version="${BUILD_VERSION}" \
  io.hass.type="app" \
  io.hass.arch="${BUILD_ARCH}" \
  org.opencontainers.image.title="Spiroflex ecoVENT Simple" \
  org.opencontainers.image.source="https://github.com/xmemphis-xyz/spiroflex-vent-clear"

COPY --from=builder /ventclear /usr/bin/ventclear
COPY run.sh /run.sh

RUN chmod +x /run.sh

CMD ["/run.sh"]
