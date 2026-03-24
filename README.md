# README

## Setup
### Quickstart (MVP setup)
- Prereqs: Go 1.21+, Docker (optional for containers).
- Start the stub manifest server (serves `/v2/manifest` and static content): `go run ./stub/manifest-server`
- Run the edge service locally against the stub (defaults to http://localhost:8080): `go run ./cmd/winnow-edge`
- Downloaded content and state are written to `/tmp/winnow/` by default.

### Docker
- Build and run both services: `sudo docker compose up --build`
- Stub manifest server: exposed on host port 8080.
- Edge service: uses `MANIFEST_URL=http://stub-manifest:8080/v2/manifest`
    - persists data to a named volume `winnow-data` mounted at `/tmp/winnow`.

### Configuration (env vars)
- `MANIFEST_URL` (default `http://localhost:8080/v2/manifest`)
- `DEVICE_TOKEN` (optional header for manifest requests)
- `POLL_INTERVAL` (default `30s`)
- `HTTP_TIMEOUT` (default `10s`)
- `DATA_DIR` (default `/tmp/winnow`)
- `STATE_FILE` (default `/tmp/winnow/state.json`)
- `WORKER_COUNT` (default `4`)
- `DOWNLOAD_TIMEOUT` (default `15s` per item)
- `MAX_ATTEMPTS` (default `3` per item)

## Requirements
### Functional
- [x] poll the manifest endpoint for changes
- [x] process changes to the manifest
- [x] download new content to the device
- [x] Implement a dummy MQTT event publisher that writes the payload to stdout.
    - Example: `Publishing {"action": "ADDED", "key": "icon-1.png"}`

### Non functional
- [x] devices can be turned off at any time by kitchen staff
    - Some kind of persistence and error recovery if poweroff mid process
- [x] there are thousands of devices in the fleet
    - Don't be too chatty
- [x] new content types can be added to the manifest service
    - Allow for changes in content types/ folders
- [x] stale content is preferable to no content
    - Error recovery should just assume previous safe/stable state
- [x] a stub manifest server will be useful for testing
    - Allows for end-to-end and fuzz tests
- [x] containerize
    - says it will run in container on device
- [x] preference for Go

### Assumptions
- expired content is fine if there are errors in the replacing process or no new version
    - partial manifest download is better than no manifest download
- ETags are unique
- docker is container engine
- new manifests get generated at most at 30s intervals
- no external dependencies allowed
    - addressed in [todos](./TODO.md)
- IoT device is not storage space constrained
    - addressed in [todos](./TODO.md)
- IoT device is only requesting content it should have access to
    - addressed in [todos](./TODO.md)
- Downloaded content is complete and correct
    - addressed in [todos](./TODO.md)

## Design/ Architecture
### System design
- Centralized server that controls a fleet of IoT clients
- The client periodically poll server for updates
    - Instead of a webhook/ callback approach to conserve edge device resources

### Key algorithm/ flow
- Do a state machine loop
    - Download manifest > Download items > Update stable manifest ETag
- Restart loop until all downloads are successful
    - Or attempt timeout is reached and see if newer manifest available

### Key components
- Main program:
    - Supervisor: State machine with timer to do polling (main program)
    - Workers: Fixed number go routines doing manifest and content download (for now manually set)
- Storage:
    - Blob storage: filesystem in `/tmp/winnow/`
        - `stable/{item type}/{blob item}`
        - `{manifest ETag}/{item type}/{blob name}`
    - Running state and ETag cache (json) (also in `/tmp/winnow/`)
        - Per manifest ETag
        - Per item/content ETag blob
- Observability:
    - slog: built in structured logs

### Dependencies
- Unsure what the dependency policy is so only golang stdlib is used
