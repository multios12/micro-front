# Seeds

seedとは、開発・確認用のテストデータをDBに投入するための機能である。
テストデータは、`./seeds` 配下に置かれた、JSON ファイルを読み込む。

## seed機能の起動方法
seeds機能は、 `seed`サブコマンドで実行できる。

```sh
go run ./cmd/micro-front seed
go run ./cmd/micro-front seed --profile default
go run ./cmd/micro-front seed --seed-dir ./seeds/default
go run ./cmd/micro-front seed --reset=false
```

## seedサブコマンド

| 引数         | 既定値    | 説明 |
| ------------ | --------- | ------------------------------------------------ |
| `--profile`  | `default` | `seeds/{profile}` を seed ディレクトリとして使う |
| `--seed-dir` | 空        | 任意の seed ディレクトリを直接指定する |
| `--reset`    | `true`    | 投入前に既存のブログ・画像を削除する |

## テストデータの種類

- `site.json`: サイト設定
- `blogs.json`: 記事データ
- `images.json`: 画像メタデータとコピー元ファイル
- `content/`: 長めの Markdown 本文
- `files/images/`: seed 時に `data/images/{blog_id}/{image_id}.png` へコピーする PNG

`blogs.json` の `content_file` は seed ディレクトリからの相対パスを指定する。
`use_sample_content: true` を指定すると `docs/sample-md.md` を本文として使う。
