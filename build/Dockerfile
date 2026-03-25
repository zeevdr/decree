# Build stage
FROM golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /bin/central-config-service ./cmd/server

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/central-config-service /bin/central-config-service

USER nonroot:nonroot

ENTRYPOINT ["/bin/central-config-service"]
