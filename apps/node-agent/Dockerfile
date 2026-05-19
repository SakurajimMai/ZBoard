# Build stage
FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/zboard-agent ./cmd/zboard-agent

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=build /out/zboard-agent /usr/local/bin/zboard-agent

ENV ZBOARD_AGENT_CONFIG_FILE=/etc/zboard-agent/runtime.json
ENV ZBOARD_AGENT_WORK_DIR=/var/lib/zboard-agent

VOLUME ["/etc/zboard-agent", "/var/lib/zboard-agent"]

ENTRYPOINT ["/usr/local/bin/zboard-agent"]
CMD ["--config", "/etc/zboard-agent/agent.env"]
