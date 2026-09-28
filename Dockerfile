FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/switchllm ./cmd/switchllm

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/switchllm /usr/local/bin/switchllm
COPY switchllm.yaml /etc/switchllm/switchllm.yaml
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/switchllm"]
CMD ["start", "--config", "/etc/switchllm/switchllm.yaml"]
