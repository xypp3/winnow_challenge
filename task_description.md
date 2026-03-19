# Task description

At Winnow we have a fleet of IoT waste tracking devices placed in kitchens around the world. We want these devices to be able to store customer specific data and to automatically update the stored data when it changes. 

There is an existing cloud service that provides an endpoint to read a manifest of the different content available for download to a device. The Open API specification for the endpoint is provided below.

Your challenge is to write a new service that will run in a container on the device and:
- poll the manifest endpoint for changes
- process changes to the manifest
- download new content to the device

An MQTT event should be published to notify interested services of newly downloaded content. For this challenge implement a dummy publisher that writes the payload to stdout. Here is an example logged event:
`Publishing {"action": "ADDED", "key": "icon-1.png"}`

Please implement the new service using whichever language you are most comfortable with, although we encourage solutions written in Go. You are permitted to use LLM coding agents for this challenge, but bear in mind that your implementation will be discussed in a technical interview. However, please ensure that you are prepared to clearly explain the choices that you have made when designing and implementing this service.

Some things to consider during implementation:
- devices can be turned off at any time by kitchen staff
- there are thousands of devices in the fleet
- new content types can be added to the manifest service
- stale content is preferable to no content
- a stub manifest server will be useful for testing

We are looking for a well structured solution that can be built and run locally. Instructions should be provided on how to do this. When running locally, the service should assume that:
- a manifest server is running at http://localhost:8080 
- other services will be reading downloaded files at /tmp/winnow. 

Documentation and tests are expected, but they should be representative rather than exhaustive. We expect this test to take approximately 2-4 hours. If you are short of time, we encourage you to summarise improvements you would make to the service in a TODO.md file.

## Submission
Your solution can be shared with us as a zip file attached to an email, or as a link to a source code repository such as Github.

## Cloud Manifest Service
This is the openapi definition for the manifest cloud endpoint:
```openapi
openapi: 3.0.3
info:
  title: Manifest API
  description: 'Manifest API used to orchestrate device content manifest information so devices know when and from where to download content'
  version: 1.0.0

paths:
  /v2/manifest:
    get:
      tags:
        - manifest-v2
      summary: Get manifest
      operationId: getManifestV2
      parameters:
        - name: "X-Authorization-Device"
          in: "header"
          description: >
            Device token to drive access control to this API as well as identify the device
          schema:
            type: "string"
        - name: "If-None-Match"
          in: "header"
          description: >
            ETag of the currently downloaded manifest (if any)
      responses:
        200:
          description: "successful operation"
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/DeviceContentManifestResponseV2"
        304:
          description: >
            The current version of the content based on If-None-Match headers is up to date,
            so no need for a cache refresh on the client-side.
        401:
          description: >
            The device token was rejected
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ApiError"
        500:
          description: >
            The device content manifest service encountered a critical error
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ApiError"

components:
  schemas:
    ApiError:
      type: object
      properties:
        errors:
          type: array
          items:
            $ref: '#/components/schemas/ErrorResponse'
    ErrorResponse:
      type: object
      properties:
        message:
          type: string
        key:
          type: string
        source:
          type: string
      required:
        - key
        - message
        - source

    DeviceContentManifestResponseV2:
      type: object
      properties:
        menus:
          $ref: '#/components/schemas/ManifestItem'
        icons:
          $ref: '#/components/schemas/ManifestItem'
    ManifestItem:
      type: object
      properties:
        unavailable:
          type: boolean
          description: >
            if true, indicates that the service providing the relevant content should be considered temporarily unavailable.
            The device should not expire unavailable content.
        items:
            type: array
            items:
              type: object
              $ref: '#/components/schemas/ContentItem'
    ContentItem:
      type: object
      properties:
        unavailable:
          type: boolean
          description: >
            if true, indicates that the specific endpoint for this item should be considered temporarily unavailable. All other fields except for name will be empty.
        name:
          type: string
          description: >
            used to identify a particular item within a type of content, e.g. current menu or en-GB translations
        uri:
          type: string
          description: The URI of the location where the content can be downloaded
        expiresAt:
          type: string
          description: The validity date the content availability expires
        ETag:
          type: string
          description: The ETag used for cache invalidation on the client side
```
