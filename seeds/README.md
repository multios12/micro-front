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
| `--reset`    | `true`    | 投入前に既存のブログ・画像・カテゴリを削除する |

## テストデータの種類

- `site.json`: サイト設定
- `categories.json`: 親見出し・小見出しと表示順（省略可能）
- `blogs.json`: 記事データ
- `images.json`: 画像メタデータとコピー元ファイル
- `content/`: 長めの Markdown 本文
- `files/images/`: seed 時に `data/images/{blog_id}/{image_id}.jpg` へ変換して保存する画像

`blogs.json` の `content_file` は seed ディレクトリからの相対パスを指定する。
`use_sample_content: true` を指定すると `docs/sample-md.md` を本文として使う。

## カテゴリのテストデータ

`categories.json` は `name`・`category_key`・`sort_order` を指定する。
小見出しは `parent_key` に親見出しの `category_key` を指定する。
`is_heading` の省略時は親なしを親見出しとして保存する。親なしの小見出しを定義する場合は、`parent_key` を省略して `is_heading: false` を指定する。
親も同じファイルに定義する。記事の `category`・`parent_category` には登録済みの表示名を指定する。必要な親・カテゴリは `categories.json` に事前定義する。記事保存時の自動登録は行わない。シード記事では同じ `categories.json` の `category_key` による指定も受け付け、表示名に変換して保存する。通常の記事APIでは表示名を指定する。

`category_key`・`parent_key` はシードファイル内だけの参照名であり、DBには保存しない。再投入では親・見出し種別・表示名で照合して既存IDを維持し、表示順を更新する。名前の変更や親の移動は管理画面で行う。
JSON内の定義順には依存せず、同じキーのカテゴリが存在する場合はIDを維持して更新する。

default プロファイルには次の2階層を用意している。

```text
お知らせ
開発メモ
  Markdown
  画像
設計
  導線
運用
  公開作業
```

記事ID 2・3・4・13は小見出しに関連付けている。
`go run ./cmd/micro-front seed` で投入後、管理画面の `Categories` で確認できる。
既定の投入操作は既存データを削除するため、確認用DBで実行する。
`--reset=false` は削除を省略するが、既存と同じID・タイトルの記事がある場合は記事投入が失敗するため、既存データへの再投入用ではない。
