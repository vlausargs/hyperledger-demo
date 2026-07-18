FROM golang:1.25-alpine AS builder
# No `apk add` here: `apk` fetches from dl-cdn.alpinelinux.org hang
# intermittently from inside a build container on this host. git is not
# needed — go mod download pulls public modules via the Go module proxy
# (HTTPS), not VCS. GOPROXY is pinned explicitly so no `direct`/VCS fallback
# is attempted.
ENV GOPROXY=https://proxy.golang.org,direct GOFLAGS=-mod=mod
WORKDIR /app
COPY packages/api/go.mod packages/api/go.sum ./
RUN go mod download
COPY packages/api/ ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.20
# golang:1.25-alpine already ships a populated CA bundle (/etc/ssl/certs) —
# reuse it from the builder stage instead of a fresh `apk add
# ca-certificates` fetch, which was observed to hang indefinitely against
# dl-cdn.alpinelinux.org from inside a build container on this host (the
# base image pull itself, via the daemon's own registry client, works
# fine — it's specifically an apk-fetch-from-inside-a-container-during-
# build issue).
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
