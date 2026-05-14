FROM golang:1.23-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o /out/telestas .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app && \
    mkdir -p /data && chown app:app /data
USER app
WORKDIR /app
COPY --from=build /out/telestas /app/telestas
VOLUME ["/data"]
ENV DB_PATH=/data/tasks.db
ENTRYPOINT ["/app/telestas"]
