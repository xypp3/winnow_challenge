FROM golang:1.21 AS build
WORKDIR /app

COPY go.mod ./
COPY . .

RUN go build -o /out/winnow-edge ./cmd/winnow-edge

FROM debian:bookworm-slim
RUN useradd -m appuser
USER appuser
WORKDIR /home/appuser

COPY --from=build /out/winnow-edge /usr/local/bin/winnow-edge

ENV DATA_DIR=/tmp/winnow \
    STATE_FILE=/tmp/winnow/state.json \
    MANIFEST_URL=http://localhost:8080/v2/manifest \
    HTTP_TIMEOUT=10s \
    POLL_INTERVAL=30s

ENTRYPOINT ["/usr/local/bin/winnow-edge"]
