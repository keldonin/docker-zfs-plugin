FROM golang:1.13-alpine AS builder

WORKDIR /plugin

ADD go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install

CMD ["/go/bin/docker-zfs-plugin"]

FROM alpine
RUN apk update && apk add zfs zfs-lts
RUN mkdir -p /run/docker/plugins /mnt/state
COPY --from=builder /go/bin/docker-zfs-plugin .
CMD ["docker-zfs-plugin"]
