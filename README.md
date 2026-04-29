# micro-front

ブログ機能を中心とした小規模な個人サイトを作成するためのWebアプリです。

管理画面で作成した記事をもとに、公開用 HTML の作成ができ、次のような特徴を持ちます。

- トップページ、記事一覧、カテゴリ一覧、記事詳細、About ページの静的 HTML 出力
- 出力する静的 HTMLには、JavaScriptを含めない
- Markdown による本文編集
- 画像のアップロード
- 管理画面からの記事作成・更新・削除
- JSON ベースでのテストデータ投入(seed機能)


## 実行方法(管理画面)

micro-front の管理画面は Web アプリです。
docker 上での実行を想定して設計しましたが、ローカルで動作させることも可能です。
必要に応じて、環境変数でパラメータを変更してください。

ビルド済みのバイナリをカレントディレクトリに置いた場合、下記のコマンドで実行できます。

```sh
./micro-front
```

Docker で動かす場合は、`docs/Dockerfile.sample` を参考にdockerfileを作成してください。
サンプルを使用して試す場合、下記のコマンドで実行できます。

```sh
docker build -f docs/Dockerfile.sample -t micro-front-run .
docker run --rm -p 3000:3000 -v "$(pwd)/data:/app/data" micro-front-run
```

| 環境変数              | 既定値            | 説明 |
| --------------------- | ----------------- | --- |
| `PORT`                | `:3000`           | Webサーバ(管理画面)の待ち受けアドレス |
| `STATIC_EXPORT_DIR`   | `./data/publish`  | 公開 HTML の出力先ディレクトリ |
| `DATA_DIR`            | `./data`          | 管理画面データの保存先ディレクトリ |
| `DB_PATH`             | `DATA_DIR/app.db` | DBファイル パス |
| `TOP_PAGE_BLOG_LIMIT` | `20`              | トップページの新着記事件数 |
| `BLOGS_PER_PAGE`      | `20`              | 記事一覧の1ページあたり件数 |

ローカルで起動した場合、下記のページからアクセスできます。
※ 既定ではローカルポート `:3000` で起動します。

```
http://localhost:3000
```

## 公開用HTMLの作成

管理画面のダッシュボードから、 `公開` ボタンを押してください。

記事単位で公開したい場合は、記事編集ページで、 `この記事を公開する` ボタンを押します。


## サブコマンド `publish`

公開用の静的 HTML を生成するための機能です。

管理画面上から公開はできますが、コマンドラインから実行したい場合に使用します。

```sh
./bin/micro-front publish
./bin/micro-front publish --target all
./bin/micro-front publish --target blog --blog-id 1
./bin/micro-front publish --publish-dir ./data/publish
```

| 引数            | 既定値              | 説明 |
| --------------- | ------------------- | ------------------------------------------- |
| `--target`      | `all`               | 出力対象。`all`, `index`, `blogs`, `blog`, `about` |
| `--blog-id`     | `0`                 | `--target blog` で使用する記事ID            |
| `--publish-dir` | `STATIC_EXPORT_DIR` | 公開用 HTML の出力先                          |

`target` の意味:

| target  | 内容                                   |
| ------- | -------------------------------------- |
| `all`   | すべてのHTMLを生成                     |
| `index` | トップページだけ生成                   |
| `blogs` | 記事一覧、記事詳細、カテゴリ一覧を生成 |
| `blog`  | 指定した記事詳細だけ生成               |
| `about` | About ページだけ生成                   |


## サブコマンド `seed`

JSON と Markdown からデータを DB に投入します。
詳しくは、`seeds/README.md` を参照してください。


## ドキュメント・設計書

- 開発時の起動・ビルド・コマンド引数: [docs/development.md](docs/development.md)
- seed 機能の使用方法・データの編集方法: [seeds/README.md](seeds/README.md)
- Markdown サンプル: [docs/sample-md.md](docs/sample-md.md)
- 公開サイトモック: [docs/mocks/public/](docs/mocks/public/)
- 管理画面モック: [docs/mocks/admin/](docs/mocks/admin/)
- 基本設計書: [docs/design/1.site-design.md](docs/design/1.site-design.md)
- 詳細設計書(共通): [docs/design/2-1.detail-common.md](docs/design/2-1.detail-common.md)
- 詳細設計書(管理画面): [docs/design/2-2.detail-admin.md](docs/design/2-2.detail-admin.md)
- 詳細設計書(公開HTML): [docs/design/2-3.detail-public.md](docs/design/2-3.detail-public.md)
