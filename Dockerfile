FROM golang:1.26.2-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

RUN test "$(go list -m)" = "github.com/kreove/omni-infra-provider-vergeos" \
    && go list -find \
       ./internal/pkg/provider/meta \
       ./internal/pkg/provider/resources

RUN go test ./...

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/omni-infra-provider-vergeos \
    ./cmd/omni-infra-provider-vergeos

FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="Omni Infrastructure Provider for VergeOS"
LABEL org.opencontainers.image.description="Community Sidero Omni infrastructure provider for VergeOS"
LABEL org.opencontainers.image.source="https://github.com/kreove/omni-infra-provider-vergeos"
LABEL org.opencontainers.image.licenses="MPL-2.0"

COPY --from=build \
    /out/omni-infra-provider-vergeos \
    /omni-infra-provider-vergeos

ENTRYPOINT ["/omni-infra-provider-vergeos"]
