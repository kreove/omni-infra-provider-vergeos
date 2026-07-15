# syntax=docker/dockerfile:1

FROM golang:1.26.2-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/omni-infra-provider-vergeos ./cmd/omni-infra-provider-vergeos

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/omni-infra-provider-vergeos /omni-infra-provider-vergeos
ENTRYPOINT ["/omni-infra-provider-vergeos"]
