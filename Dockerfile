FROM golang:alpine as builder

RUN apk update && \
    apk add build-base

RUN mkdir /build

ADD . /build/

WORKDIR /build

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static" -s -w' -o main .

RUN echo "compiled file size: $(du -sh main)"
RUN sleep 1

FROM debian:buster

WORKDIR /app

ENV SERVE_ROOT /wwwroot
ENV SERVE_PORT 6969
ENV SERVE_DIR false

COPY --from=builder /build/main /app/

CMD ["./main",
        "-port", "${SERVE_PORT}",
        "-root", "${SERVE_PORT}",
        "-servedir", "${SERVE_DIR}"]