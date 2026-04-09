FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/owui-memory-daemon ./cmd/daemon

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /out/owui-memory-daemon /app/owui-memory-daemon

ENTRYPOINT ["/app/owui-memory-daemon"]

