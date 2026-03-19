# Stub Manifest Server

Minimal server that mimics the cloud manifest endpoint and serves static content for local testing.

- Endpoint: `GET /v2/manifest` (returns manifest JSON and `ETag`; honors `If-None-Match`)
- Content: served from `/content/*` (files embedded under `content/`)

## Run
- Local: `go run ./stub/manifest-server`
- Docker: `docker build -f stub/manifest-server/Dockerfile . -t manifest-stub && docker run -p 8080:8080 manifest-stub`

## Env vars
- `PORT` (default `8080`)
- `BASE_URL` (default `http://localhost:PORT`)
- `MANIFEST_ETAG` (default `W/"demo-1"`)
