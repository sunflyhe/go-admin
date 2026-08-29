# API 多阶段构建。运行时完全由环境变量配置，不依赖未打包的 config.yaml。
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
  && addgroup -S app \
  && adduser -S -G app app \
  && mkdir -p /data/uploads \
  && chown -R app:app /data
WORKDIR /app
COPY --from=build /out/api /app/api
USER app
EXPOSE 8080
ENTRYPOINT ["/app/api", "-config", ""]
