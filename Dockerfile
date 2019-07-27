FROM golang:1-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build
COPY go.mod go.sum /build/
RUN go get

COPY . /build
RUN go build -o /usr/bin/maumirror

FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /usr/bin/maumirror

VOLUME /data
USER 1337:1337

CMD ["/usr/bin/maumirror"]
