# access-report

CaddyのJSONアクセスログから、レポートHTMLファイルを作成します。
HTMLファイルには、記事アクセス数・ユニークアクセス数・クローラー・404・スキャン疑いを出力します。

## レポートを作成する

実行ファイル `access-report` を用意して実行します。クローラー判定ルールファイルが存在しない場合は、標準のルールで自動作成します。以下は、実行ファイルを `bin/` に配置した例です。相対パスはコマンドを実行するディレクトリが基準になります。

```sh
./bin/access-report --input /var/log/caddy/access.log.json --host example.com
```

処理が完了したら、`data/access-report/index.html` をブラウザーで開いて確認します。出力先は `--output` で変更できます。

ローリング済みログも自動探索する場合：

```sh
./bin/access-report \
  --log-dir /var/log/caddy \
  --log-name access.log.json \
  --host example.com
```

## 引数と環境変数

優先順位は **コマンドライン引数 ＞ 環境変数 ＞ 既定値** です。引数を省略した場合だけ環境変数を読み込みます。

| 引数 | 環境変数 | 既定値 | 内容 |
| --- | --- | --- | --- |
| `--input` | `ACCESS_REPORT_INPUT` | なし | 入力ログ。複数回指定可能 |
| `--log-dir` | `ACCESS_REPORT_LOG_DIR` | なし | ログ自動探索ディレクトリ |
| `--log-name` | `ACCESS_REPORT_LOG_NAME` | `access.log.json` | 現行ログ名 |
| `--host` | `ACCESS_REPORT_HOST` | なし | 対象ホスト。必須 |
| `--state` | `ACCESS_REPORT_STATE` | `./data/access-report-state/state.db` | 集計状態DB |
| `--output` | `ACCESS_REPORT_OUTPUT` | `./data/access-report/index.html` | 出力HTML |
| `--crawler-rules` | `ACCESS_REPORT_CRAWLER_RULES` | `./config/access-report-crawlers.json` | クローラー判定ルールJSON |
| `--blog-db` | `ACCESS_REPORT_BLOG_DB` | `./data/app.db` | 記事タイトル取得用DB |
| `--help` / `-h` | なし | — | ヘルプ表示 |

- `--input` または `--log-dir` のどちらかが必須で、環境変数の反映後も同時指定はできません。位置引数は使用できません。
- `ACCESS_REPORT_INPUT` は改行区切り（LF・CRLF対応、空行は無視）で複数ファイルを指定できます。`--input` を指定すると、環境変数の入力一覧全体を置き換えます。
- 環境変数の空文字も設定値として扱います。`--blog-db ""` または `ACCESS_REPORT_BLOG_DB=""` で記事DB参照を無効化できます。
- 記事DBの既定ファイルが存在しない場合は記事ID表示になります。引数・環境変数で指定した記事DBが存在しない場合や読み取りに失敗した場合はエラーです。

環境変数を使う例：

```sh
export ACCESS_REPORT_HOST=example.com
export ACCESS_REPORT_LOG_DIR=/var/log/caddy
./bin/access-report
```

## 集計と運用

同じ状態DBを指定して再実行すると、追加されたログだけを集計に加え、レポートを更新します。未取り込みの退避ログが削除される前に定期実行してください。ログの内容確認には、取り込み済みの部分も読み込みます。

クローラー判定は `config/access-report-crawlers.json` で管理します。`--crawler-rules` または `ACCESS_REPORT_CRAWLER_RULES` で保存先を変更できます。ファイルがない場合は親ディレクトリも含めて自動作成します。既存ファイルは上書きせず、内容が不正な場合はエラーにします。各ルールの `name` を一覧へ表示し、複数一致時は先に書かれた有効ルールを優先します。名前だけの変更は再集計不要です。判定条件を変更した場合は、別の `--state` を指定して元ログから再集計します。記事のアクセス数・ユニークアクセス数からは、UAルールで判定したクローラーを除外します。ユニーク判定はIP＋User-Agentの組み合わせです。DBと同じ場所に作成する `<DBファイル名>.visitor.key` も、DBと一緒にバックアップしてください。

集計タイムゾーンは `Asia/Tokyo` です。全期間は取り込み済みログの最初の日から最後の日まで、過去1か月は最後の日を基準に表示します。タイムゾーンは実行時には変更できません。

スキャン疑い一覧は、同じIPから1日に100件以上の404かつ20種類以上のパスがある場合に表示します。この判定条件は実行時には変更できません。一覧にはアクセス元IPをそのまま表示します。

ビルド・テスト・集計設定の変更については[開発者向けガイド](DEVELOPMENT.md)を参照してください。

