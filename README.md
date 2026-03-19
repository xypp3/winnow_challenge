# README

## Requirements
### Functional
- [ ] poll the manifest endpoint for changes
- [ ] process changes to the manifest
- [ ] download new content to the device

### Non functional
- [ ] devices can be turned off at any time by kitchen staff
    - Some kind of persistence and error recovery if poweroff mid process
- [ ] there are thousands of devices in the fleet
    - Don't be too chatty
- [ ] new content types can be added to the manifest service
    - Allow for changes in content types/ folders
- [ ] stale content is preferable to no content
    - Error recovery should just assume previous safe/stable state
- [ ] a stub manifest server will be useful for testing
    - Allows for end-to-end and fuzz tests
- [ ] containerize
    - says it will run in container on device
- [ ] preference for Go

## Discussion
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
    - ETag cache KeyValue DB (also in `/tmp/winnow/`
        - Per manifest ETag
        - Per item/content ETag blob
- Observability:
    - slog: built in structured logs
    - Prometheus: standard metrics service
        - unsure if appropriate for IoT but

### Dependencies
- bolt, KV database written in Golang
- Prometheus client, to link to prometheus observability tooling
