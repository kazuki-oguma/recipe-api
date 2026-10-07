APP_NAME   := recipe-api
IMAGE_NAME := recipe-api
IMAGE_TAG  := latest
PORT       := 8080

# docker-run時にコンテナへ渡すDB接続情報（必要に応じて `make docker-run DB_HOST=...` で上書きする）
DB_HOST     ?= host.docker.internal
DB_PORT     ?= 5432
DB_USER     ?= postgres
DB_PASSWORD ?=
DB_NAME     ?= recipes_db
DB_SSLMODE  ?= disable

.PHONY: build run test clean docker-build docker-run docker-stop

## ローカル用にGoバイナリをビルドする
build:
	go build -o bin/$(APP_NAME) .

## ローカルでビルドして起動する
run: build
	./bin/$(APP_NAME)

## テストを実行する
test:
	go test ./...

## ビルド成果物を削除する
clean:
	rm -rf bin

## Dockerイメージをビルドする
docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

## Dockerコンテナを起動する（起動時にAPIサーバーが自動的に立ち上がる）
docker-run:
	docker run --rm -p $(PORT):$(PORT) \
		-e PORT=$(PORT) \
		-e DB_HOST=$(DB_HOST) \
		-e DB_PORT=$(DB_PORT) \
		-e DB_USER=$(DB_USER) \
		-e DB_PASSWORD=$(DB_PASSWORD) \
		-e DB_NAME=$(DB_NAME) \
		-e DB_SSLMODE=$(DB_SSLMODE) \
		--name $(APP_NAME) \
		$(IMAGE_NAME):$(IMAGE_TAG)

## 起動中のコンテナを停止する
docker-stop:
	docker stop $(APP_NAME)
