# Build stage
FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/zboard-agent ./cmd/zboard-agent

# Download Xray and sing-box binaries
FROM alpine:3.20 AS runtime-dl
RUN apk add --no-cache curl unzip tar

ARG XRAY_VERSION=25.5.16
ARG SINGBOX_VERSION=1.11.8
ARG TARGETARCH=amd64

RUN curl -fsSL "https://github.com/XTLS/Xray-core/releases/download/v${XRAY_VERSION}/Xray-linux-64.zip" \
      -o /tmp/xray.zip && \
    unzip /tmp/xray.zip -d /tmp/xray && \
    chmod +x /tmp/xray/xray

RUN curl -fsSL "https://github.com/SagerNet/sing-box/releases/download/v${SINGBOX_VERSION}/sing-box-${SINGBOX_VERSION}-linux-${TARGETARCH}.tar.gz" \
      -o /tmp/singbox.tar.gz && \
    tar -xzf /tmp/singbox.tar.gz -C /tmp && \
    mv /tmp/sing-box-${SINGBOX_VERSION}-linux-${TARGETARCH}/sing-box /tmp/sing-box && \
    chmod +x /tmp/sing-box

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata iptables ip6tables

WORKDIR /app
COPY --from=build /out/zboard-agent /usr/local/bin/zboard-agent
COPY --from=runtime-dl /tmp/xray/xray /usr/local/bin/xray
COPY --from=runtime-dl /tmp/sing-box /usr/local/bin/sing-box

ENV ZBOARD_AGENT_CONFIG_FILE=/etc/zboard-agent/runtime.json
ENV ZBOARD_AGENT_WORK_DIR=/var/lib/zboard-agent
ENV ZBOARD_AGENT_RUNTIME_BINARY=/usr/local/bin/xray
ENV TZ=Asia/Shanghai

VOLUME ["/etc/zboard-agent", "/var/lib/zboard-agent"]

ENTRYPOINT ["/usr/local/bin/zboard-agent"]
CMD ["--config", "/etc/zboard-agent/agent.env"]
