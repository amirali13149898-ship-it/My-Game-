FROM golang:alpine AS build
WORKDIR /src
COPY . .
# `go mod tidy` resolves modernc.org/sqlite + pgx and writes go.sum if it is not committed yet.
# golang:alpine (latest) because the newest driver versions need a newer Go than 1.24.
ENV GOTOOLCHAIN=auto
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ardian ./cmd/ardian

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/ardian /usr/local/bin/ardian
COPY --from=build /src/assets /assets
# Mount a Railway Volume at /data so the SQLite file survives deploys.
ENV DB_PATH=/data/ardian.db ASSETS_DIR=/assets
# Keep the footprint tiny on free tiers: one OS thread of Go scheduling, GC earlier, soft memory cap.
ENV GOMAXPROCS=1 GOGC=50 GOMEMLIMIT=100MiB
CMD ["ardian"]
