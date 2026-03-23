FROM golang:1.21 AS build
WORKDIR /app
COPY go.mod .
COPY stub/manifest-server ./stub/manifest-server
RUN go build -o /out/manifest-stub ./stub/manifest-server

FROM debian:bookworm-slim
RUN useradd -m stub
COPY --from=build /out/manifest-stub /tmp/manifest-stub
RUN install -m 0755 -o stub -g stub /tmp/manifest-stub /usr/local/bin/manifest-stub && rm /tmp/manifest-stub
USER stub
WORKDIR /home/stub
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/manifest-stub"]
