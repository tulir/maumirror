FROM golang:1-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build
COPY go.mod go.sum /build/
RUN go get

COPY . /build
RUN go build -o /usr/bin/maumirror

FROM alpine:latest

RUN apk add --no-cache ca-certificates git openssh-client bash

COPY --from=builder /usr/bin/maumirror /usr/bin/maumirror

VOLUME /data
VOLUME /config
EXPOSE 29321

RUN useradd -u 29321 maumirror
USER 29321:29321

CMD ["/usr/bin/maumirror", "-c", "/config/config.json"]
