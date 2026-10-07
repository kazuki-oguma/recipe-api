# ---- ビルドステージ ----
FROM golang:1.22-alpine AS builder

WORKDIR /src

# 依存解決をソースコピーより先に行い、依存関係が変わらない限りレイヤーキャッシュを効かせる
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# pgxは純Go実装でCGO不要なため、CGO_ENABLED=0で静的バイナリをビルドする
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/recipe-api .

# ---- 実行ステージ ----
FROM alpine:3.20

# PostgreSQLへのTLS接続（DB_SSLMODE=require等）に必要なCA証明書と、ログ表示用のタイムゾーンデータ
RUN apk add --no-cache ca-certificates tzdata

# 非rootユーザーで実行する
RUN addgroup -S app && adduser -S app -G app
USER app

WORKDIR /app
COPY --from=builder /out/recipe-api ./recipe-api

EXPOSE 8080

# コンテナ起動時にAPIサーバーを起動する
ENTRYPOINT ["./recipe-api"]
