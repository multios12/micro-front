# 開発手順

micro-front の開発時に使う起動方法、ビルド方法、コマンドライン引数をまとめる。

## 起動方法

### Go サーバを起動

```sh
go run ./cmd/micro-front
```

既定では `:3001` で起動する。

```text
http://localhost:3001/
```

管理画面の静的ファイルは `web/static/` から配信される。

### フロント開発サーバを起動

```sh
yarn --cwd front dev
```

Vite の開発サーバは通常 `http://127.0.0.1:3000` で起動する。

### VS Code で起動

VS Code のデバッグ構成を使う場合は `Dev: Front + API` を実行する。

- フロント: `http://127.0.0.1:3000`
- API: `http://localhost:3001`

設定ファイル:

- [../.vscode/launch.json](../.vscode/launch.json)
- [../.vscode/tasks.json](../.vscode/tasks.json)

## ビルド方法

### 管理画面をビルド

```sh
yarn --cwd front build
```

`front/package.json` の `postbuild` により、ビルド結果は `web/static/` へコピーされる。
Go サーバ単体で管理画面を配信したい場合は、このコマンドを実行してから起動する。

### Go のテスト

```sh
go test ./...
```

環境によって Go の既定キャッシュディレクトリが書き込めない場合は、次のように `/tmp` を使う。

```sh
GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache go test ./...
```

## コマンドライン引数

`cmd/micro-front` はサブコマンドなしで HTTP サーバを起動する。

```sh
go run ./cmd/micro-front
```

### `publish`

公開用の静的 HTML を生成する。

```sh
go run ./cmd/micro-front publish
go run ./cmd/micro-front publish --target all
go run ./cmd/micro-front publish --target blog --blog-id 1
go run ./cmd/micro-front publish --publish-dir ./data/publish
```

引数:

| 引数 | 既定値 | 説明 |
| --- | --- | --- |
| `--target` | `all` | 出力対象。`all`, `index`, `blogs`, `blog`, `about` |
| `--blog-id` | `0` | `--target blog` または一部の `blogs` 再生成で使う記事ID |
| `--publish-dir` | `STATIC_EXPORT_DIR` | 静的 HTML の出力先 |

`target` の意味:

| target | 内容 |
| --- | --- |
| `all` | トップ、記事一覧、記事詳細、カテゴリ一覧、About、Error を生成 |
| `index` | トップページだけ生成 |
| `blogs` | 記事一覧、記事詳細、カテゴリ一覧を生成 |
| `blog` | 指定した記事詳細だけ生成 |
| `about` | About ページだけ生成 |

### `seed`

JSON と Markdown からテストデータを DB に投入する。

```sh
go run ./cmd/micro-front seed
go run ./cmd/micro-front seed --profile default
go run ./cmd/micro-front seed --seed-dir ./seeds/default
go run ./cmd/micro-front seed --reset=false
```

引数:

| 引数 | 既定値 | 説明 |
| --- | --- | --- |
| `--profile` | `default` | `seeds/{profile}` を seed ディレクトリとして使う |
| `--seed-dir` | 空 | 任意の seed ディレクトリを直接指定する |
| `--reset` | `true` | 投入前に既存のブログ・画像を削除する |

seed データの編集方法は [../seeds/README.md](../seeds/README.md) を参照する。

## 環境変数

| 環境変数 | 既定値 | 説明 |
| --- | --- | --- |
| `ADDR` | `:3001` | Go サーバの待ち受けアドレス |
| `ADMIN_STATIC_DIR` | `web/static` | 管理画面の静的ファイル配信元 |
| `STATIC_EXPORT_DIR` | `./data/publish` | 公開 HTML の出力先 |
| `DATA_DIR` | `./data` | SQLite DB と画像保存先 |
| `DB_PATH` | `DATA_DIR/app.db` | SQLite DB パス |
| `TOP_PAGE_BLOG_LIMIT` | `20` | トップページの新着記事件数 |
| `BLOGS_PER_PAGE` | `20` | 記事一覧の1ページあたり件数 |

## よく使う流れ

seed データを投入して公開 HTML を生成する。

```sh
go run ./cmd/micro-front seed
go run ./cmd/micro-front publish
```

管理画面をビルドして Go サーバで確認する。

```sh
yarn --cwd front build
go run ./cmd/micro-front
```
