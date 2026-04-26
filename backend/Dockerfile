FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /krasis-server ./cmd/server

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /krasis-server /usr/local/bin/
COPY --from=builder /etc/ssl/certs /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo/

RUN adduser -D -g '' krasis
USER krasis

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/krasis-server"]
