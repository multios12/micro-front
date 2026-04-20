# micro-front

小規模なブログ機能を中心にした、管理画面付きの静的 HTML 公開サイトです。

管理画面でサイト設定・記事・画像を編集し、`publish` によって公開用 HTML を `data/publish/` 配下へ生成します。

## 主な機能

- 管理画面からの記事作成・更新・削除
- Markdown による本文編集
- 記事画像のアップロードと公開用画像コピー
- トップページ、記事一覧、カテゴリ一覧、記事詳細、About ページの静的 HTML 出力
- JSON ベースの seed データ投入

## ディレクトリ概要

```text
cmd/micro-front/        アプリケーションのエントリポイント
internal/               Go 側のアプリケーション実装
front/                  管理画面の Svelte/Vite プロジェクト
web/static/             管理画面のビルド済み静的ファイル
internal/web/templates/ 公開ページ生成用テンプレート
data/                   SQLite DB、アップロード画像、公開HTML出力先
seeds/                  seed コマンド用の編集可能なテストデータ
docs/                   設計書、モック、開発手順
```

## ドキュメント

- 開発時の起動・ビルド・コマンド引数: [docs/development.md](docs/development.md)
- seed データの編集方法: [seeds/README.md](seeds/README.md)
- Markdown サンプル: [docs/sample-md.md](docs/sample-md.md)
- 公開サイトモック: [docs/mocks/public/](docs/mocks/public/)
- 管理画面モック: [docs/mocks/admin/](docs/mocks/admin/)

## ビルド方法

管理画面をビルドし、Go の配信用静的ファイルとして `web/static/` へコピーする。

```sh
yarn --cwd front build
```

Go アプリケーションのバイナリを作成する。

```sh
go build -o ./bin/micro-front ./cmd/micro-front
```

## ビルド後バイナリの実行方法

HTTP サーバを起動する。

```sh
./bin/micro-front
```

既定では `:3001` で起動する。

```text
http://localhost:3001/
```

seed データを投入して公開 HTML を生成する場合は、次の順で実行する。

```sh
./bin/micro-front seed
./bin/micro-front publish
./bin/micro-front
```

## 引数

サブコマンドなしの場合は HTTP サーバを起動する。

```sh
./bin/micro-front
```

### `publish`

公開用の静的 HTML を生成する。

```sh
./bin/micro-front publish
./bin/micro-front publish --target all
./bin/micro-front publish --target blog --blog-id 1
./bin/micro-front publish --publish-dir ./data/publish
```

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
./bin/micro-front seed
./bin/micro-front seed --profile default
./bin/micro-front seed --seed-dir ./seeds/default
./bin/micro-front seed --reset=false
```

| 引数 | 既定値 | 説明 |
| --- | --- | --- |
| `--profile` | `default` | `seeds/{profile}` を seed ディレクトリとして使う |
| `--seed-dir` | 空 | 任意の seed ディレクトリを直接指定する |
| `--reset` | `true` | 投入前に既存のブログ・画像を削除する |

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

## 設計書

- [docs/design/1.site-design.md](docs/design/1.site-design.md)
- [docs/design/2-1.detail-common.md](docs/design/2-1.detail-common.md)
- [docs/design/2-2.detail-admin.md](docs/design/2-2.detail-admin.md)
- [docs/design/2-3.detail-public.md](docs/design/2-3.detail-public.md)
