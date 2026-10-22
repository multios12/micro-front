# access-report 開発者向けガイド

利用方法・引数・環境変数・運用方法は [README](README.md) を参照してください。

## ビルドとテスト

リポジトリのルートで実行します。リポジトリで指定されたGo環境を使用してください。

```sh
go build -o ./bin/access-report ./cmd/access-report
go test ./cmd/access-report
```

サンプルログを使ってレポートを作成する例：

```sh
./bin/access-report --input sample/access.log.json --host modern-utopia.net
```

## 集計・表示設定の変更

[main.go](main.go) 冒頭に設定定数をまとめています。

| 定数 | 値 | 用途 |
| --- | --- | --- |
| `reportTimezone` | `Asia/Tokyo` | 日別集計のタイムゾーン |
| `scanMin404` | `100` | スキャン疑いとする1日あたりの最小404件数 |
| `scanMinPaths` | `20` | スキャン疑いとする1日あたりの最小パス種類数 |

変更後は再ビルドが必要です。タイムゾーンの変更は集計条件の変更になるため、別の状態DBに元ログを取り込み直してください。スキャン判定のしきい値は表示条件なので、再取り込みは不要です。

## レポートのテンプレート

`templates/` のテンプレートはバイナリに埋め込まれます。変更後は再ビルド・再実行して出力HTMLを確認してください。元の画面モックは [docs/mocks/access-report/index.html](../../docs/mocks/access-report/index.html) です。

## 既定のクローラー判定ルール

[templates/access-report-crawlers.json](templates/access-report-crawlers.json) をバイナリに埋め込み、指定先にルールファイルがない場合に書き出します。既定ルールを変更した場合は再ビルドが必要です。既存のルールファイルは自動更新しません。

## 設計資料

- [アクセスレポート設計](../../docs/access-report.md)
- [テーブル設計](../../docs/access-report-tables.md)
