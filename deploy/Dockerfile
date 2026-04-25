# Single multi-stage Dockerfile shared by all three Havoc binaries.
# `docker build --build-arg BIN=havoc-agent` selects which one to ship.
# Keeping one file means the toolchain version, build flags, and base
# image stay in lock-step across the trio.

ARG GO_VERSION=1.26
FROM golang:${GO_VERSION}-alpine AS build
ARG BIN
WORKDIR /src

# Cache module downloads in a layer of their own — the bulk of the
# build time on a clean machine.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Static binary so we can land it in a distroless image.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/${BIN} ./cmd/${BIN}

FROM gcr.io/distroless/static-debian12:nonroot
ARG BIN
COPY --from=build /out/${BIN} /app
USER nonroot
ENTRYPOINT ["/app"]
