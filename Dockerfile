# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.22-alpine AS build

WORKDIR /src

# Cache dependencies first for faster rebuilds.
COPY go.mod go.sum* ./
RUN go mod download

# Build a static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /shorty .

# ---- Runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /
COPY --from=build /shorty /shorty

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/shorty"]
