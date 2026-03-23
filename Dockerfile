FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -X main.version=2.0.0" \
    -o moderation-server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=builder /build/moderation-server /moderation-server

VOLUME ["/logs"]

EXPOSE 8080

ENV TZ=Asia/Shanghai

ENTRYPOINT ["/moderation-server"]
