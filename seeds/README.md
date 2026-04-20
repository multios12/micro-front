# Seeds

開発・確認用のテストデータを投入するための JSON ファイルを置く。

```sh
go run ./cmd/micro-front seed
go run ./cmd/micro-front seed --profile default
go run ./cmd/micro-front seed --seed-dir ./seeds/default
go run ./cmd/micro-front seed --reset=false
```

- `site.json`: サイト設定
- `blogs.json`: 記事データ
- `images.json`: 画像メタデータとコピー元ファイル
- `content/`: 長めの Markdown 本文
- `files/images/`: seed 時に `data/images/{blog_id}/{image_id}.png` へコピーする PNG

`blogs.json` の `content_file` は seed ディレクトリからの相対パスを指定する。
`use_sample_content: true` を指定すると `docs/sample-md.md` を本文として使う。
