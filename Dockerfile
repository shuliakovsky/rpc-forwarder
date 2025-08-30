FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

ARG VERSION="0.0.1"
ARG COMMIT_HASH=""

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH GOARM=$GOARM \
    go build -ldflags "-X main.Version=${VERSION} -X main.CommitHash=${COMMIT_HASH}" \
    -o /out/rpc-forwarder ./cmd/app


FROM alpine:3.22

RUN addgroup -S rpcforwarder && adduser -S rpcforwarder -G rpcforwarder

WORKDIR /app

COPY --from=builder /out/rpc-forwarder .
COPY configs/networks ./configs/networks

RUN chown -R rpcforwarder:rpcforwarder /app

USER rpcforwarder

EXPOSE 8080

CMD ["./rpc-forwarder"]