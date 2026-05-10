# 開発手順

micro-front の開発に関する情報をまとめます。

## ディレクトリ概要

```text
cmd/micro-front/        アプリケーションのエントリポイント
internal/               Go 側のアプリケーション実装
front/                  管理画面の Svelte/Vite プロジェクト
internal/web/admin.html 管理画面のビルド後 HTML
internal/web/templates/ 公開ページ生成用テンプレート
seeds/                  編集可能なテストデータ(seed コマンドによる投入用)
docs/                   ドキュメント（設計書、モック、開発手順）
```

## ビルド方法

### 管理画面をビルド

```sh
yarn --cwd front build
```

`front/package.json` の `postbuild` により、ビルド結果は `internal/web/admin.html` へコピーされる。
このファイルは Go サーバで必須資材として参照されるため、`front` をビルドしたあとに内容を確認してからコミットする。
Go サーバ単体で管理画面を配信したい場合は、このコマンドを実行してから起動する。


### Go サーバをビルド

```sh
go build ./cmd/micro-front
```

必要に応じて出力先を指定する場合は、`-o` を付けてください。

```sh
go build -o ./bin/micro-front ./cmd/micro-front
```

## デバッグ実行

### VS Code で起動

VS Code のデバッグ構成を使う場合は `Dev: Front + API` を実行します。

- フロント: `http://localhost:3000`
- API: `http://localhost:3001`

設定ファイル:

- [../.vscode/launch.json](../.vscode/launch.json)
- [../.vscode/tasks.json](../.vscode/tasks.json)


### Go サーバを起動

VS Codeのデバッグ構成を使用しない場合、Go サーバとフロント開発用サーバを別々に起動する必要があります。

```sh
go run ./cmd/micro-front
```
 ※既定では `:3000` で起動します。

```text
http://localhost:3000/
```

管理画面のHTMLは `data/admin.html` を優先して公開します。
`data/admin.html` が存在しない場合は、バイナリに埋め込まれたフォールバックHTMLを返します。
管理画面の静的ファイルは `web/static/` から配信されます。

### フロント開発サーバを起動

```sh
yarn --cwd front dev
```

Vite の開発サーバは通常 `http://127.0.0.1:3000` で起動します。
管理 API は同一オリジンの `/admin/api` を相対URLで呼び出します。
開発サーバは `vite.config.ts` の proxy を通して Go 側へ転送します。

## Go のテスト

```sh
go test ./...
```

環境によって Go の既定キャッシュディレクトリが書き込めない場合は、次のように `/tmp` を使う。

```sh
GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache go test ./...
```

## GitHub Releases での公開

このリポジトリでは、タグを作成すると GitHub Actions がアプリケーションの配布物を作成し、GitHub Releases にアップロードします。

公開手順は次のとおりです。

1. `main` ブランチに必要な変更をマージします
1. バージョンタグを作成して push します
   ```sh
   git tag -a v0.8.5 -m ''; git push origin --tags
   ```
1. GitHub の Releases ページで、Actions が作成したリリース資材を確認します

Release には、各 OS 向けの `micro-front` バイナリを含めます。必要に応じて、管理画面のビルド成果物や公開 HTML を別途添付する運用にも拡張できます。

## 環境変数(ビルド用)

| 環境変数                  | 既定値 | 説明 |
| ------------------------- | ------ | --- |
| なし | - | 管理 API は相対URLで呼び出す |

## テストデータ投入

seed機能を使用して、テストデータを投入できます。

```sh
go run ./cmd/micro-front seed
go run ./cmd/micro-front publish
```
