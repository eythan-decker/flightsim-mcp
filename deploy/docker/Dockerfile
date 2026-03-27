FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /flightsim-mcp ./cmd/flightsim-mcp

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /flightsim-mcp /flightsim-mcp
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/flightsim-mcp"]
