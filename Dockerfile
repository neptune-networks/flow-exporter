FROM golang:alpine as builder
ARG LDFLAGS=""

RUN apk --update --no-cache add git build-base gcc

COPY . /build
WORKDIR /build

RUN go mod download
RUN go build -ldflags "${LDFLAGS}" -o flow-exporter cmd/flow-exporter/main.go

FROM alpine:latest
ARG src_dir

RUN apk update --no-cache
COPY --from=builder /build/flow-exporter /

EXPOSE 9590
ENTRYPOINT ["./flow-exporter"]
