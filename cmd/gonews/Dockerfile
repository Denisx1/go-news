FROM golang:1.26.2-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /app/gonews ./cmd/gonews

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata chromium font-liberation \
    && adduser -D -u 1000 app
ENV TZ=Europe/Kyiv \
    DATA_DIR=/app/data
WORKDIR /app
COPY --from=builder /app/gonews /app/gonews
RUN mkdir -p /app/data && chown -R app:app /app
USER app
CMD ["./gonews"]