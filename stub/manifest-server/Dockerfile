FROM golang:1.21 AS build
WORKDIR /app
COPY ../../go.mod ./
COPY . .
RUN go build -o /out/manifest-stub .

FROM debian:bookworm-slim
RUN useradd -m stub
USER stub
WORKDIR /home/stub
COPY --from=build /out/manifest-stub /usr/local/bin/manifest-stub
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/manifest-stub"]
