# Build stage: cgo is required for the SQLite driver.
FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /techtoss .

# Runtime stage.
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /techtoss /app/techtoss
COPY template ./template
COPY static ./static

# Install air for development
FROM golang:1.22-bookworm AS dev
RUN go install github.com/cosmtrek/air@v1.52.0
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY template ./template
COPY static ./static

EXPOSE 2000
CMD ["air", "-c", ".air.toml"]