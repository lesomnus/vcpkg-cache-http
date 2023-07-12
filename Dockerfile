# syntax=docker/dockerfile:1
FROM golang:1.20-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN go build -o "a"


FROM scratch

COPY --from=builder "/app/a" "/vcpkg-cache-http"

ENTRYPOINT ["/vcpkg-cache-http"]

EXPOSE 15151/tcp
VOLUME ["/vcpkg-cache"]
