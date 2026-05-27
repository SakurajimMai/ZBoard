# Build stage
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates

ARG TARGETOS
ARG TARGETARCH
ARG GIT_COMMIT=dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN test -n "${TARGETOS}" && test -n "${TARGETARCH}" && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags "-s -w -X github.com/zboard/api-server/internal/buildinfo.Commit=${GIT_COMMIT}" -o /out/zboard-api ./cmd/server

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget tzdata
WORKDIR /app

COPY --from=build /out/zboard-api /usr/local/bin/zboard-api

ENV ZBOARD_HOST=0.0.0.0 \
    ZBOARD_PORT=3000 \
    ZBOARD_DB_DIALECT=sqlite \
    ZBOARD_DB_PATH=/data/zboard.sqlite \
    TZ=Asia/Shanghai

VOLUME ["/data"]
EXPOSE 3000

ENTRYPOINT ["/usr/local/bin/zboard-api"]
