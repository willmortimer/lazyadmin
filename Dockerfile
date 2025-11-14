# Stage 1: build static Go binary
FROM golang:1.23-alpine AS builder

RUN adduser -D -g '' builduser
USER builduser

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w" \
  -o /home/builduser/lazyadmin ./cmd/lazyadmin

# Stage 2: minimal runtime
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /home/builduser/lazyadmin /app/lazyadmin

ENTRYPOINT ["/app/lazyadmin"]

