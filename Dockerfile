FROM golang:1.21-bookworm as builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . ./

RUN go build -v -o ambrosio

FROM debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/ambrosio /app/ambrosio

# Run the web service on container startup.
CMD ["/app/ambrosio"]

# [END run_helloworld_dockerfile]
# [END cloudrun_helloworld_dockerfile]
