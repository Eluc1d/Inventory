# Build stage: cgo is required for the SQLite driver.
FROM golang:1.22-bookworm AS build
WORKDIR /src
# The build context (Inventory/) doesn't include the repo's .git, so the
# version has to come in from outside — pass it explicitly, e.g.:
#   docker build --build-arg APP_VERSION=$(git describe --tags --always --dirty) .
# Left unset, the binary falls back to detecting it at runtime (see version.go).
ARG APP_VERSION=dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -ldflags "-X main.version=${APP_VERSION}" -o /techtoss .

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
# The dev container bind-mounts the repo's real .git (see docker-compose.yml)
# so version.go can shell out to `git describe`. Git refuses to operate on a
# repo it doesn't consider "safe" when the mounted directory's owning UID
# doesn't match the container user (common with bind mounts) — mark it
# trusted explicitly rather than relying on that matching by luck.
RUN git config --global --add safe.directory /app
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY template ./template
COPY static ./static

EXPOSE 2000
CMD ["air", "-c", ".air.toml"]