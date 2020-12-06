FROM golang:alpine as builder
RUN apk update && \
    apk add build-base
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static" -s -w' -o simpleserv .

FROM debian:buster
WORKDIR /app
COPY --from=builder /build/simpleserv /app
ENTRYPOINT ["/app/simpleserv"]