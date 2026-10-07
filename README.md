# recipe-api

レシピ情報を管理するREST APIサーバー（Go実装）。`create.sql` のPostgreSQLスキーマ（`recipes` テーブル）を操作する。

API仕様はOpenAPI 3.0形式で [`openapi.yaml`](./openapi.yaml) にまとめている。Swagger UI等で確認する場合は例えば以下のように表示できる。

```sh
npx @redocly/cli preview-docs openapi.yaml
```

## 必要環境

- Go 1.22以上
- PostgreSQL（`create.sql` は `SERIAL` / `TIMESTAMP` / トリガーなどPostgreSQL固有の構文を使用）

## セットアップ

1. PostgreSQLにデータベースを作成し、`create.sql` を適用する。

   ```sh
   createdb recipes_db
   psql -d recipes_db -f create.sql
   ```

   `create.sql` では、MySQLの `AUTO_INCREMENT` 相当として `SERIAL` を、
   `ON UPDATE CURRENT_TIMESTAMP` 相当として `updated_at` を自動更新するトリガーを使用している。

2. 環境変数を設定する（未設定時は括弧内のデフォルト値を使用）。

   | 変数名 | 説明 | デフォルト |
   | --- | --- | --- |
   | `DB_HOST` | DBホスト | `127.0.0.1` |
   | `DB_PORT` | DBポート | `5432` |
   | `DB_USER` | DBユーザー | `postgres` |
   | `DB_PASSWORD` | DBパスワード | 空文字 |
   | `DB_NAME` | DB名 | `recipes_db` |
   | `DB_SSLMODE` | SSL接続モード（`disable`/`require`等） | `disable` |
   | `PORT` | APIサーバーの待受ポート | `8080` |

3. 依存関係を解決し、サーバーを起動する。

   ```sh
   go mod tidy
   go run .
   ```

## Docker

Makefile経由でDockerイメージのビルド・起動ができる。コンテナ起動時にAPIサーバーが自動的に立ち上がる。

```sh
# イメージをビルドする
make docker-build

# コンテナを起動する（DB_HOST等はmake実行時に上書き可能。例: make docker-run DB_HOST=db）
make docker-run

# コンテナを停止する
make docker-stop
```

`docker-run` はデフォルトで `DB_HOST=host.docker.internal` を使用し、コンテナからホストマシン上のPostgreSQLへ接続する想定になっている（Docker Desktop環境向け）。別のホストやdocker-composeのサービス名を使う場合は `DB_HOST` を上書きすること。

Makefileにはローカルビルド用のターゲットも用意している。

```sh
make build  # bin/recipe-api を生成
make run    # ビルドして実行
make test   # go test ./... を実行
```

## テスト

ハンドラ層の正常系・異常系・境界値テストを用意している（`internal/handler/recipe_handler_test.go`）。
DBアクセス層はフェイク実装（インメモリのインターフェース実装）に差し替えてテストするため、テスト実行にDB接続は不要。

```sh
go test ./...
```

> 本プロジェクトはGoで実装しているため、テストはGo標準の `testing` パッケージを使用している（JUnitはJava向けのフレームワークのため適用不可）。

## エンドポイント一覧

ベースURL: `http://localhost:8080`

| メソッド | パス | 概要 |
| --- | --- | --- |
| POST | `/recipes` | レシピを新規作成する |
| GET | `/recipes` | レシピ一覧を取得する |
| GET | `/recipes/{id}` | 指定IDのレシピを1件取得する |
| PATCH | `/recipes/{id}` | 指定IDのレシピを部分更新する |
| DELETE | `/recipes/{id}` | 指定IDのレシピを削除する |

上記以外の未定義パスへアクセスした場合は `404 Not Found` を返す（`net/http` 標準ルーターの挙動）。
また、定義済みパスに対して許可されていないHTTPメソッドでアクセスした場合は `405 Method Not Allowed` を返す（Go 1.22以降の `http.ServeMux` の標準挙動）。

### ステータスコードの方針

公開済みエンドポイント（`/recipes`, `/recipes/{id}`）へのアクセスは、リクエスト内容の正当性に関わらず常に`200 OK`を返す。
`404 Not Found`は「未定義のエンドポイントへのアクセス」と「指定IDのリソースが存在しない場合」のみに限定する
（`test/test.rest.js` の公式テストスイートがPOSTの必須項目不足ケースで200を期待しているため、この方針に合わせている）。

| ケース | ステータス |
| --- | --- |
| 正常に処理できた場合 | `200 OK` |
| リクエストの必須項目不足・形式不正（POST/PATCH）、idが数値でない場合 | `200 OK`（bodyの`message`でエラーを通知） |
| 指定IDのリソースが存在しない場合（GET/PATCH/DELETE） | `404 Not Found` |
| 未定義のエンドポイントへのアクセス | `404 Not Found` |
| サーバー内部エラー | `500 Internal Server Error` |

### POST /recipes

リクエスト例:

```json
{
  "title": "チキンカレー",
  "making_time": "45分",
  "serves": "4人",
  "ingredients": "玉ねぎ,肉,スパイス",
  "cost": 1000
}
```

成功時（200）:

```json
{
  "message": "Recipe successfully created!",
  "recipe": [
    {
      "id": 3,
      "title": "チキンカレー",
      "making_time": "45分",
      "serves": "4人",
      "ingredients": "玉ねぎ,肉,スパイス",
      "cost": "1000",
      "created_at": "2026-10-07 12:00:00",
      "updated_at": "2026-10-07 12:00:00"
    }
  ]
}
```

必須項目不足時（400）。`required` は不足項目に関わらず必須項目の固定一覧をカンマ区切りの文字列で返す:

```json
{
  "message": "Recipe creation failed!",
  "required": "title, making_time, serves, ingredients, cost"
}
```

### GET /recipes

データベース内の全レシピを返す。一覧表示では `created_at`/`updated_at` を含まない省略表現を返す。

成功時（200）:

```json
{
  "recipes": [
    { "id": 1, "title": "チキンカレー", "making_time": "45分", "serves": "4人", "ingredients": "玉ねぎ,肉,スパイス", "cost": "1000" },
    { "id": 2, "title": "オムライス", "making_time": "30分", "serves": "2人", "ingredients": "玉ねぎ,卵,スパイス,醤油", "cost": "700" },
    { "id": 3, "title": "トマトスープ", "making_time": "15分", "serves": "5人", "ingredients": "玉ねぎ, トマト, スパイス, 水", "cost": "450" }
  ]
}
```

失敗時（500、DBアクセスエラーなど。POSTと同様に `message` フィールドのみを返す。GETには必須項目チェックが無いため `required` は含まない）:

```json
{ "message": "レシピ一覧の取得に失敗しました" }
```

### GET /recipes/{id}

成功時（200）。`recipe`は1件の配列で、`created_at`/`updated_at`は含まない:

```json
{
  "message": "Recipe details by id",
  "recipe": [
    { "id": 1, "title": "チキンカレー", "making_time": "45分", "serves": "4人", "ingredients": "玉ねぎ,肉,スパイス", "cost": "1000" }
  ]
}
```

存在しない場合（404）:

```json
{ "message": "recipe not found" }
```

### PATCH /recipes/{id}

リクエストボディは `title`, `making_time`, `serves`, `ingredients`, `cost`（更新したい項目のみ指定する部分更新。未指定の項目は変更されない）。

リクエスト例:

```json
{ "cost": 1200 }
```

成功時（200）。`recipe`は1件の配列で、`id`・`created_at`・`updated_at`は含まない:

```json
{
  "message": "Recipe successfully updated!",
  "recipe": [
    { "title": "チキンカレー", "making_time": "45分", "serves": "4人", "ingredients": "玉ねぎ,肉,スパイス", "cost": "1200" }
  ]
}
```

### DELETE /recipes/{id}

成功時（200）:

```json
{ "message": "Recipe successfully removed!" }
```

存在しない場合（404）:

```json
{ "message": "No Recipe found" }
```

## 設計メモ（REST架構制約への準拠）

[restfulapi.net](https://restfulapi.net/rest-architectural-constraints/) の各制約への対応は以下の通り。

- **クライアント/サーバー分離**: APIはJSONのみを返し、UIの関心事を一切含まない。
- **ステートレス**: サーバー側にセッション状態を保持せず、各リクエストは独立して処理される（DB接続プール以外の状態を持たない）。
- **キャッシュ可能性**: `GET` はべき等かつ安全なためキャッシュ可能。将来的にキャッシュヘッダ（`ETag`/`Cache-Control`）を付与する余地を残している。
- **統一インターフェース**: `/recipes` をリソースの名詞として扱い、操作はHTTPメソッド（POST/GET/PATCH/DELETE）で表現している。レスポンスは自己記述的なJSON（`Content-Type: application/json`）。
- **階層化システム**: ハンドラ層・リポジトリ層・DB層を分離しており、将来的にリバースプロキシやロードバランサを前段に挟んでもクライアントからは透過的。
- **コードオンデマンド（任意）**: 本APIはデータのみを返却し、実行可能コードの配信は行わない（未実装で問題ない任意制約）。
