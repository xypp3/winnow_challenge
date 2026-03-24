# TODO

## Open Questions
- Do newer manifests always succeed the older ones? or do they coexist?
- e.g. if an endpoint never returns does the service keep trying to download the content forever?
    - or is there a retry limit?
- Do we wanna persist old content and manifests for rollbacks?
- How do we do manifest server discovery?

## If dependencies are allowed
- For running state and etags: **sqlite DB**
    - better performance and reliability as number of blobs grows
    - also good choice for embedded applications
- For more metrics: **prometheus client**
    - Allow for instrumentation of application + resource metrics

## Features
- Use protobufs if manifests are large or often
- Use a UDP based protocol for content download
- A old content/blob removal mechanism to manage device storage space
- Exponential backoff with jitter for workers and fetch

## Security
- Use TLS for https
- Use asymentric keys to verify that this IoT device is the authorized client for a specific endpoint/content
- Content hashes to verify downloaded content is not corrupted

## Testing
- Build out fuzztesting
    - Introduce network "failures" from server stub
    - Introduce better assertions/ failing conditions to increase visibility of error cases
