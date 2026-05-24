FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY packages/api/go.mod packages/api/go.sum ./
RUN go mod download
COPY packages/api/ ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
