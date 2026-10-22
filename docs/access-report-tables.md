# アクセスレポート テーブル設計案

## 前提

[画面サンプル](mocks/access-report/index.html)と[基本設計](access-report.md)に対応するSQLiteの論理・物理設計案。DDLは `cmd/access-report/schema.sql` に実装し、初回CLI実行時に専用DBへ適用する。

- micro-frontの記事DBとは別の `data/access-report-state/state.db` を使用する。
- 1つのDBは1ホスト・1組の集計条件を扱う。条件変更時はログから別DBへ再構築する。
- 生ログを保存せず、画面に必要な日別集計とユニーク判定情報、404アクセス元IP、取り込み状態を保存する。
- 集計値はログの保存期限と独立して保持する。初期仕様では自動削除しない。
- 期間、順位、割合、平均、日別ユニーク数は表示時に計算する。月別・全期間の専用テーブルは作らない。
- この文書のテーブル設計を、基本設計にある暫定的な保存項目の具体化とする。

## 画面とテーブルの対応

| 画面 | 主に参照するテーブル | 計算 |
| --- | --- | --- |
| 観測期間 | article_daily | 記事アクセスがある最小・最大日付 |
| クローラーを除くアクセス数 | article_daily | 選択期間のaccess_count合計 |
| ユニークアクセス数 | article_daily_visitors | 選択期間のvisitor_keyの重複排除件数 |
| アクセスのあった記事数 | article_daily | 選択期間のarticle_idの重複排除件数 |
| 1日あたりのアクセス | article_daily | アクセス合計÷表示期間の日数（0件の日を含む） |
| 日別グラフ・表 | article_daily / article_daily_visitors | 日別アクセス合計・日別visitor_key重複排除件数 |
| 記事ランキング | articles / article_daily / article_daily_visitors | 記事別アクセス合計・記事別visitor_key重複排除件数 |
| クローラーアクセス数・一覧 | crawler_agents / crawler_daily | User-Agent別・期間別アクセス合計 |
| 404件数・一覧 | not_found_paths / not_found_daily | リクエストパス別・期間別アクセス合計 |
| 差分取り込み | report_state / log_generations / log_files | 条件照合・世代照合・再開位置管理 |

記事ランキングと各一覧の上位20件・21件目以降は、同じ結果セットをHTML生成側で分ける。アコーディオン用テーブルは不要。

## 型・日付・期間の共通ルール

- 日付は集計タイムゾーンでの `YYYY-MM-DD` のTEXTとする。
- 処理日時はUTCのRFC 3339文字列とする。ログの観測日時はUnix時刻のマイクロ秒をINTEGERで保存する。
- 件数、オフセットは0以上のINTEGER。記事IDは正のINTEGER。
- ハッシュはSHA-256またはHMAC-SHA-256の32バイトBLOB。
- 通常列は原則NOT NULL。NULLを許す項目は各表に明示する。
- 接続ごとに外部キー制約を有効にする。集計値の更新と確定オフセット更新は同じトランザクションに含める。

過去1か月は、レポート基準日の翌日0時を終端として暦で1か月遡った区間とする。月末は遡り先の月の最終日に丸める。サンプルでは2026-09-16以上、2026-10-16未満。基準日は初期仕様では対象ホストの有効ログの最大日付とし、実行日の変更だけで表示が変わらないようにする。全期間はDBに取り込んだ全履歴を対象とする。

アクセスレポートの観測期間は選択期間内の実アクセスの最小・最大日付、日別グラフの表示範囲と平均の分母は選択した期間とする。クローラー・404の期間も同じ基準日で計算する。該当0件の観測期間は「該当なし」とする。未来日時などを有効ログとして受け付ける範囲は実装前に確定する。

## テーブル一覧

### report_state：DB全体の状態

1行のみ。ホスト・集計条件・スキーマとアクセス元識別鍵の対応を管理する。

| 列 | 型 | 内容 |
| --- | --- | --- |
| id | INTEGER PK | 常に1 |
| schema_version | INTEGER | スキーマの版 |
| rules_version | INTEGER | 集計ロジックの版 |
| host | TEXT | 正規化した対象ホスト |
| timezone | TEXT | 例：Asia/Tokyo |
| config_json | TEXT | 集計条件の正規化JSON。秘密情報は含めない |
| config_hash | BLOB | config_jsonのSHA-256 |
| visitor_key_id | TEXT | 外部保存したHMAC鍵の識別子 |
| revision | INTEGER | 入力ファイルのコミットごとに増加 |
| created_at / updated_at | TEXT | 作成・更新日時 |

ルール版、ホスト、タイムゾーン、HTTP条件、URL正規化、除外条件、クローラー判定、文字列上限、IP・User-Agent正規化を条件照合に含める。クローラー判定は外部JSONの有効ルールのID・pattern・記載順を正規化したハッシュで照合する。外部ファイルのパス、name、descriptionは集計条件に含めない。表示期間・出力先・デザインは含めない。revisionはHTMLのmeta要素へ記録し、画面に管理情報欄は追加しない。

### log_generations：論理的なログ世代

ファイルのrename・圧縮・コピーをまたいで追跡する単位。パスやinodeを主キーにしない。

| 列 | 型 | 内容 |
| --- | --- | --- |
| id | INTEGER PK | 内部の世代ID |
| committed_offset | INTEGER | 展開後の確定バイト位置。最後に処理した完全行の改行直後 |
| committed_sha256 | BLOB、NULL可 | [0, committed_offset)のハッシュ。offset=0のときのみNULL |
| head_length / tail_length | INTEGER | 部分照合の対象バイト数 |
| head_sha256 / tail_sha256 | BLOB、NULL可 | 先頭・確定位置直前の部分ハッシュ |
| sealed | INTEGER | 0：追記可能、1：退避済みで完全取り込みを確認 |
| first_ts_us / last_ts_us | INTEGER、NULL可 | 対象ホストの有効ログの最小・最大日時。行順には依存しない |
| valid_lines / invalid_lines | INTEGER | 完全行の有効・不正件数 |
| host_out_lines | INTEGER | 対象ホスト以外の有効行 |
| article_access_lines | INTEGER | クローラーを除く記事アクセス集計件数 |
| article_excluded_lines / article_other_lines | INTEGER | 対象ホスト内の除外・記事集計対象外件数 |
| unidentified_article_lines | INTEGER | IP欠落などでユニーク判定できない記事アクセス。article_access_linesの内数 |
| oversized_dimension_lines | INTEGER | UA・404パスの表示文字列を省略した行。分類件数の内数 |
| created_at / updated_at | TEXT | 作成・最終コミット日時 |

valid_lines = host_out_lines + article_access_lines + article_excluded_lines + article_other_linesとする。クローラー・404は記事集計と並行して評価し、この分類には加算しない。

sealedは現行ファイルの一時的なEOFだけでは立てない。退避済みの安定した入力について完全行を末尾まで取り込み、gzipなら整合性検証まで完了してから立てる。未完行が残る退避ログは警告し、完了扱いにしない。

### log_files：ログ世代の物理ファイル履歴

同じ世代の現行ファイル・退避ファイル・gzipを対応させる。inode変更は新規行、同一物理ファイルのrenameはパス更新とする。パス再利用があるためpathに一意制約を付けない。

| 列 | 型 | 内容 |
| --- | --- | --- |
| id | INTEGER PK | ファイル観測ID |
| generation_id | INTEGER FK | log_generations.id |
| path | TEXT | CLIから見た絶対パス。HTMLには出さない |
| compression | TEXT | none / gzip |
| device / inode | TEXT、NULL可 | OSで取得できる場合。数値幅と移植性のためTEXT |
| observed_size | INTEGER | 最終確認した物理サイズ。gzipでは圧縮後サイズ |
| first_seen_at / last_seen_at | TEXT | 初回・最終確認日時 |

device/inodeは補助情報であり、再利用やコピーがあるため単独で同一性を確定しない。コピー・圧縮はlog_generationsの既読領域ハッシュでも照合する。完全一致する別ログとの区別がつかなければ停止する。過去の完了世代の行は、退避ログ削除後も残して再取り込みを防ぐ。

### articles：記事の識別

| 列 | 型 | 内容 |
| --- | --- | --- |
| article_id | INTEGER PK | URLから取得した記事ID |
| path | TEXT UNIQUE | 正規化済みの記事パス |

記事タイトルは状態DBに保持しない。HTML生成時に任意指定の記事DBから読み取り専用で取得し、IDで対応させる。タイトルは表示用のため集計条件には含めない。記事ID表記の先頭ゼロは除去して正規化し、`/blogs/1.html` と `/blogs/01.html` を同じ記事とする。SQLite INTEGERの範囲を超えるIDは記事対象外とし、ルールに含める。

### article_daily：日別の記事アクセス数

| 列 | 型 | 内容 |
| --- | --- | --- |
| day | TEXT PKの一部 | 集計日 |
| article_id | INTEGER PKの一部・FK | articles.article_id |
| access_count | INTEGER | アクセス数 |
| unidentified_count | INTEGER | ユニーク判定不能なアクセス数。access_count以下 |

複合主キーは(day, article_id)。0件の行は作らず、画面生成時に日付列を補完する。

### article_daily_visitors：日別・記事別のアクセス元集合

| 列 | 型 | 内容 |
| --- | --- | --- |
| day | TEXT PKの一部・複合FK | article_daily.day |
| article_id | INTEGER PKの一部・複合FK | article_daily.article_id |
| visitor_key | BLOB PKの一部 | HMAC-SHA-256で生成したアクセス元識別子 |

複合主キーは(day, article_id, visitor_key)。同じ日の同じ記事に何度アクセスしても1行だけ保持する。アクセス回数はarticle_daily側で毎回増やす。

visitor_keyは、正規化したIPとUser-Agentを曖昧さのない形式（例：JSON配列）に直列化してHMAC化する。日付・記事IDはHMAC入力に含めず、複数日・複数記事をまたいで同じ識別子になるようにする。IPはclient_ipを優先し、欠落時remote_ip。IPがない場合は集合へ挿入せずunidentified_countを増やす。User-Agent欠落は空文字として扱う。

秘密鍵は状態ディレクトリの `<DBファイル名>.visitor.key` に保存し、DBには鍵そのものを入れない。鍵IDは鍵のハッシュで照合し、紛失・変更時は継続を拒否する。鍵を日替わりにすると全期間ユニーク数が求められなくなるため、自動更新しない。HMAC識別子もアクセス元を継続識別する情報として扱う。

### crawler_agents：クローラーのUser-Agent

| 列 | 型 | 内容 |
| --- | --- | --- |
| id | INTEGER PK | 内部ID |
| rule_id | TEXT | 最初に一致した有効ルールID。外部JSONで管理し、DB内の外部キーは設けない |
| ua_hash | BLOB UNIQUE | 正規化前に結合した完全なUser-Agent文字列のSHA-256 |
| display_ua | TEXT | 表示文字列。最大1,024 Unicode文字 |
| is_truncated | INTEGER | 文字列省略の有無 |

判定は完全な文字列に対して行い、集約も完全な文字列のハッシュで行う。表示省略前の文字列は保存しない。User-Agentに含まれる制御文字は表示用に置換する。大文字・小文字は判定時のみ無視し、別User-Agentとしての集約では維持する。

判定ルールは[基本設計の外部JSON案](access-report.md#クローラー判定ルール)に従う。複数パターンに一致しても重複加算しない。判定ルールそのものをDBの編集対象にせず、report_stateの条件情報で適用条件を識別する。非クローラーのUAを保存しないため、判定条件変更時の過去の再分類には元ログからの再構築が必要になる。

### crawler_daily：日別・クローラー別の件数

| 列 | 型 | 内容 |
| --- | --- | --- |
| day | TEXT PKの一部 | 集計日 |
| agent_id | INTEGER PKの一部・FK | crawler_agents.id |
| access_count | INTEGER | 対象ホストの全メソッド・パス・ステータスの該当件数 |

記事アクセスの除外設定とは独立して、対象ホストのクローラー推定アクセスを数える。

### not_found_paths：404になったパス

| 列 | 型 | 内容 |
| --- | --- | --- |
| id | INTEGER PK | 内部ID |
| path_hash | BLOB UNIQUE | クエリ文字列を除いた完全なパスのSHA-256 |
| display_path | TEXT | 表示文字列。最大2,048 Unicode文字 |
| is_truncated | INTEGER | 文字列省略の有無 |

パスの大小文字・パーセントエンコード・末尾スラッシュは維持し、異なるパスを不用意に合算しない。表示省略があってもpath_hashで区別する。制御文字は表示用に置換し、HTMLではリンク化しない。クエリ文字列はハッシュ入力にも保存値にも含めない。

### not_found_daily：日別・パス別404件数

| 列 | 型 | 内容 |
| --- | --- | --- |
| day | TEXT PKの一部 | 集計日 |
| path_id | INTEGER PKの一部・FK | not_found_paths.id |
| access_count | INTEGER | ステータス404の件数。全メソッド対象 |

クローラーによる404はcrawler_dailyとnot_found_dailyの両方に入る。両者を足してリクエスト総数としない。

## DDL案

以下は初期スキーマのDDL。実行時は埋め込んだschema.sqlを使用する。

```sql
PRAGMA foreign_keys = ON;

CREATE TABLE report_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    schema_version INTEGER NOT NULL CHECK (schema_version > 0),
    rules_version INTEGER NOT NULL CHECK (rules_version > 0),
    host TEXT NOT NULL,
    timezone TEXT NOT NULL,
    config_json TEXT NOT NULL,
    config_hash BLOB NOT NULL CHECK (length(config_hash) = 32),
    visitor_key_id TEXT NOT NULL,
    revision INTEGER NOT NULL DEFAULT 0 CHECK (revision >= 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE log_generations (
    id INTEGER PRIMARY KEY,
    committed_offset INTEGER NOT NULL DEFAULT 0 CHECK (committed_offset >= 0),
    committed_sha256 BLOB,
    head_length INTEGER NOT NULL DEFAULT 0 CHECK (head_length >= 0),
    tail_length INTEGER NOT NULL DEFAULT 0 CHECK (tail_length >= 0),
    head_sha256 BLOB,
    tail_sha256 BLOB,
    sealed INTEGER NOT NULL DEFAULT 0 CHECK (sealed IN (0, 1)),
    first_ts_us INTEGER,
    last_ts_us INTEGER,
    valid_lines INTEGER NOT NULL DEFAULT 0 CHECK (valid_lines >= 0),
    invalid_lines INTEGER NOT NULL DEFAULT 0 CHECK (invalid_lines >= 0),
    host_out_lines INTEGER NOT NULL DEFAULT 0 CHECK (host_out_lines >= 0),
    article_access_lines INTEGER NOT NULL DEFAULT 0 CHECK (article_access_lines >= 0),
    article_excluded_lines INTEGER NOT NULL DEFAULT 0 CHECK (article_excluded_lines >= 0),
    article_other_lines INTEGER NOT NULL DEFAULT 0 CHECK (article_other_lines >= 0),
    unidentified_article_lines INTEGER NOT NULL DEFAULT 0 CHECK (unidentified_article_lines >= 0),
    oversized_dimension_lines INTEGER NOT NULL DEFAULT 0 CHECK (oversized_dimension_lines >= 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK ((committed_offset = 0 AND committed_sha256 IS NULL) OR
           (committed_offset > 0 AND committed_sha256 IS NOT NULL AND length(committed_sha256) = 32)),
    CHECK (head_length <= committed_offset AND tail_length <= committed_offset),
    CHECK ((head_length = 0 AND head_sha256 IS NULL) OR
           (head_length > 0 AND head_sha256 IS NOT NULL AND length(head_sha256) = 32)),
    CHECK ((tail_length = 0 AND tail_sha256 IS NULL) OR
           (tail_length > 0 AND tail_sha256 IS NOT NULL AND length(tail_sha256) = 32)),
    CHECK ((first_ts_us IS NULL AND last_ts_us IS NULL) OR
           (first_ts_us IS NOT NULL AND last_ts_us IS NOT NULL AND first_ts_us <= last_ts_us)),
    CHECK (valid_lines = host_out_lines + article_access_lines + article_excluded_lines + article_other_lines),
    CHECK (unidentified_article_lines <= article_access_lines)
);

CREATE TABLE log_files (
    id INTEGER PRIMARY KEY,
    generation_id INTEGER NOT NULL REFERENCES log_generations(id),
    path TEXT NOT NULL,
    compression TEXT NOT NULL CHECK (compression IN ('none', 'gzip')),
    device TEXT,
    inode TEXT,
    observed_size INTEGER NOT NULL CHECK (observed_size >= 0),
    first_seen_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL
);
CREATE INDEX log_files_identity ON log_files(device, inode);
CREATE INDEX log_files_path ON log_files(path);
CREATE INDEX log_files_generation ON log_files(generation_id);

CREATE TABLE articles (
    article_id INTEGER PRIMARY KEY CHECK (article_id > 0),
    path TEXT NOT NULL UNIQUE
);

CREATE TABLE article_daily (
    day TEXT NOT NULL,
    article_id INTEGER NOT NULL REFERENCES articles(article_id),
    access_count INTEGER NOT NULL CHECK (access_count > 0),
    unidentified_count INTEGER NOT NULL DEFAULT 0 CHECK (unidentified_count >= 0 AND unidentified_count <= access_count),
    PRIMARY KEY (day, article_id)
) WITHOUT ROWID;

CREATE TABLE article_daily_visitors (
    day TEXT NOT NULL,
    article_id INTEGER NOT NULL,
    visitor_key BLOB NOT NULL CHECK (length(visitor_key) = 32),
    PRIMARY KEY (day, article_id, visitor_key),
    FOREIGN KEY (day, article_id) REFERENCES article_daily(day, article_id)
) WITHOUT ROWID;

CREATE TABLE crawler_agents (
    id INTEGER PRIMARY KEY,
    rule_id TEXT NOT NULL,
    ua_hash BLOB NOT NULL UNIQUE CHECK (length(ua_hash) = 32),
    display_ua TEXT NOT NULL CHECK (length(display_ua) <= 1024),
    is_truncated INTEGER NOT NULL CHECK (is_truncated IN (0, 1))
);
CREATE TABLE crawler_daily (
    day TEXT NOT NULL,
    agent_id INTEGER NOT NULL REFERENCES crawler_agents(id),
    access_count INTEGER NOT NULL CHECK (access_count > 0),
    PRIMARY KEY (day, agent_id)
) WITHOUT ROWID;

CREATE TABLE not_found_paths (
    id INTEGER PRIMARY KEY,
    path_hash BLOB NOT NULL UNIQUE CHECK (length(path_hash) = 32),
    display_path TEXT NOT NULL CHECK (length(display_path) <= 2048),
    is_truncated INTEGER NOT NULL CHECK (is_truncated IN (0, 1))
);
CREATE TABLE not_found_daily (
    day TEXT NOT NULL,
    path_id INTEGER NOT NULL REFERENCES not_found_paths(id),
    access_count INTEGER NOT NULL CHECK (access_count > 0),
    PRIMARY KEY (day, path_id)
) WITHOUT ROWID;

CREATE TABLE scan_sources (
    id INTEGER PRIMARY KEY,
    ip TEXT NOT NULL UNIQUE
);
CREATE TABLE scan_daily_paths (
    day TEXT NOT NULL,
    source_id INTEGER NOT NULL REFERENCES scan_sources(id),
    path_id INTEGER NOT NULL REFERENCES not_found_paths(id),
    access_count INTEGER NOT NULL CHECK (access_count > 0),
    first_ts_us INTEGER NOT NULL,
    last_ts_us INTEGER NOT NULL CHECK (last_ts_us >= first_ts_us),
    PRIMARY KEY (day, source_id, path_id)
) WITHOUT ROWID;
CREATE INDEX scan_daily_paths_source ON scan_daily_paths(source_id, day);
CREATE TABLE scan_daily_agents (
    day TEXT NOT NULL,
    source_id INTEGER NOT NULL REFERENCES scan_sources(id),
    ua_hash BLOB NOT NULL CHECK (length(ua_hash) = 32),
    display_ua TEXT NOT NULL CHECK (length(display_ua) <= 1024),
    rule_id TEXT NOT NULL,
    access_count INTEGER NOT NULL CHECK (access_count > 0),
    PRIMARY KEY (day, source_id, ua_hash)
) WITHOUT ROWID;
CREATE INDEX scan_daily_agents_source ON scan_daily_agents(source_id, day);
```

日付の妥当性はCLIで検証する。日付を先頭とする複合主キーで期間抽出する。article_daily_visitorsの全期間重複排除では一時的な集計処理が必要になるが、初期段階で重複する集計テーブルは増やさない。実測で必要ならアクセス元中心の索引を追加する。

## 表示用SQL例

`:from_day` は開始日、`:until_day` は終了日の翌日。全期間では日付条件を外す。集計0件は0に補完する。

### 概要・日別

```sql
SELECT COALESCE(SUM(access_count), 0) AS accesses,
       COUNT(DISTINCT article_id) AS articles,
       MIN(day) AS observed_from, MAX(day) AS observed_to
FROM article_daily
WHERE day >= :from_day AND day < :until_day;

SELECT COUNT(DISTINCT visitor_key) AS unique_accesses
FROM article_daily_visitors
WHERE day >= :from_day AND day < :until_day;

SELECT day, SUM(access_count) AS accesses
FROM article_daily
WHERE day >= :from_day AND day < :until_day
GROUP BY day ORDER BY day;

SELECT day, COUNT(DISTINCT visitor_key) AS unique_accesses
FROM article_daily_visitors
WHERE day >= :from_day AND day < :until_day
GROUP BY day ORDER BY day;
```

日別の結果を日付で合わせ、不足日を0にする。同じアクセス元が同日に複数記事を見ても、概要・日別のユニーク数では1件とする。

### 記事ランキング

```sql
WITH accesses AS (
    SELECT article_id, SUM(access_count) AS access_count
    FROM article_daily
    WHERE day >= :from_day AND day < :until_day
    GROUP BY article_id
), visitors AS (
    SELECT article_id, COUNT(DISTINCT visitor_key) AS unique_count
    FROM article_daily_visitors
    WHERE day >= :from_day AND day < :until_day
    GROUP BY article_id
)
SELECT a.article_id, a.path, c.access_count,
       COALESCE(v.unique_count, 0) AS unique_count
FROM accesses c
JOIN articles a ON a.article_id = c.article_id
LEFT JOIN visitors v ON v.article_id = c.article_id
ORDER BY c.access_count DESC, a.article_id ASC;
```

アクセス数の集計とユニーク数の集計を別々に行ってから結合し、集合の行数によってアクセス数が増幅されることを防ぐ。割合は記事アクセス数÷選択期間の総記事アクセス数。記事別ユニーク数の合計は概要のユニーク数と一致するとは限らない。

### クローラー・404

```sql
SELECT a.id, a.rule_id, a.display_ua, a.is_truncated, SUM(d.access_count) AS access_count
FROM crawler_daily d JOIN crawler_agents a ON a.id = d.agent_id
WHERE d.day >= :from_day AND d.day < :until_day
GROUP BY a.id
ORDER BY access_count DESC, a.display_ua ASC, a.id ASC;

SELECT p.id, p.display_path, p.is_truncated, SUM(d.access_count) AS access_count
FROM not_found_daily d JOIN not_found_paths p ON p.id = d.path_id
WHERE d.day >= :from_day AND d.day < :until_day
GROUP BY p.id
ORDER BY access_count DESC, p.display_path ASC, p.id ASC;
```

各結果の件数合計から概要のリクエスト数、結果行数からUser-Agent種類数・パス数を求める。表示を省略した文字列には省略記号と識別用ハッシュの短い表記を添え、同じ表示文字列の別項目を区別する。

## 差分更新の手順

1. 同じDBを同時実行しない前提で、スキーマ・条件・HMAC鍵を照合する。アプリケーションの排他ロックは設けない。
2. 入力世代を判定し、確定オフセットから新しい完全行を読む。
3. 記事行はarticlesを登録し、article_dailyを加算する。識別できるアクセス元はarticle_daily_visitorsへ重複時無視で挿入する。
4. 同じ入力行をクローラー・404の各条件でも評価し、該当するマスターを登録して日別件数を加算する。
5. 世代の処理件数・オフセット・ハッシュ、物理ファイル情報、report_state.revisionを同じトランザクションで更新する。
6. 全入力の処理成功後、2種類のレポート×2期間の結果を取得してHTMLを生成する。

全既読領域ハッシュはSHA-256の確定ダイジェストを保存し、独自のハッシュ内部状態は保存しない。差分実行でもハッシュ計算のため既読バイトを走査するが、JSON解析・DB加算は未処理部分だけ行う。これにより「全ファイルI/Oを省略する差分」にはならない。50MB単位の入力で実測し、必要なら将来チャンクハッシュ方式を検討する。

集計は世代別に分けず加算するため、任意の世代だけを後から削除・差し替えして巻き戻す機能は初期対象外。内容変更を検出した場合は停止し、全ログから別DBへ再構築する。

## 保持方針と確認事項

- 元ログの削除後も全期間ユニーク数を再現するため、article_daily_visitorsは自動削除しない。
- UA・404パスも全期間一覧の再現に必要なため、マスターと日別件数を保持する。文字列長には上限を設けるが、種類数は固定で切り捨てない。
- UAはヘッダー由来で任意の内容を含み得る。表示用文字列はHTMLエスケープし、外部リンクを作らない。
- 全期間データ保持を選ぶため、DBは履歴に応じて増える。容量を実測し、将来保存期限を設ける場合は「全期間」の意味とユニーク集計精度も同時に変更する。
- 実装前に、クライアントIPの信頼元、期間基準日、HMAC鍵の配置・バックアップ、文字列上限を最終確認する。

## 検証観点

- 同じアクセス元が複数日・複数記事へアクセスしても期間全体のユニーク数が正しい。
- IP欠落時もアクセス数は増え、ユニーク集合には入らない。
- 初回全件と複数回の差分の全テーブル集計結果が一致する。
- クローラー404は両一覧へ入り、記事アクセス数には加算されない。
- 異常終了で集計値だけ、またはオフセットだけが確定しない。
- 期間境界、0件、記事ランキングの同順位、21件目以降の出力を確認する。
- 表示上限を超えるUA・パスが他の値と誤って合算されない。
- 鍵・条件不一致を検出して、既存集計を更新せず停止する。

実装では先頭最大4KiBのハッシュをhead_length / head_sha256に保存し、同一性照合の候補絞り込みに利用する。最終判定は既読領域全体のSHA-256による。tail_length / tail_sha256は将来用の既定値（0 / NULL）のまま保持する。

集計ルール版2では、記事条件を満たすクローラーをarticle_excluded_linesへ記録し、article_daily / article_daily_visitors / articlesへは追加しない。クローラー・404は独立して集計を継続する。

クローラー一覧はrule_idから現在の外部JSONのnameを取得して併記する。nameをDBへ複写せず、名前だけの変更で再集計を必要としない。複数一致は有効ルールの記載順で最初を採用するため、優先順位やIDの変更には再構築が必要。

## スキャン疑い用テーブル（スキーマ版3）

ユーザー指定によりアクセス元IPをそのまま表示・保存する。記事のユニーク集合は従来どおりHMAC識別子で管理する。以下は404だけを対象に、既存のnot_found_daily更新と同じトランザクションで保存する。

| テーブル | 主キー・列 | 用途 |
| --- | --- | --- |
| scan_sources | id INTEGER PK、ip TEXT UNIQUE NOT NULL | 正規化したIP。UAが変わっても同じアクセス元 |
| scan_daily_paths | PK(day, source_id, path_id)、access_count、first_ts_us、last_ts_us | IP・日付・パス別404件数と最小・最大観測日時。source_idとpath_idは外部キー |
| scan_daily_agents | PK(day, source_id, ua_hash)、display_ua、rule_id、access_count | IP・日付・UA別404件数。表示UAは最大1,024文字、完全UAのSHA-256で集約。rule_idはクローラー未判定なら空文字 |

source_idとdayの索引でアクセス元の期間別内訳を抽出する。パスはnot_found_pathsを共用し、クエリ文字列を保存しない。無効・欠落IPの行は上記3テーブルに入れず、従来の404総数には含める。IP有無で記事やクローラーの判定を変更しない。

判定は日別・source_id別にSUM(access_count)とCOUNT(path_id)を求め、両方のしきい値を満たした日を持つアクセス元を選ぶ。選択期間全体のSUM、COUNT(DISTINCT path_id)、最小・最大時刻を一覧に表示する。しきい値は保存値を変えないのでconfig_hashに含めない。日付のタイムゾーンは集計条件に含める。

追加検証：同一IPのUA変更、同一パスへの連打、99/100件と19/20種類の境界、日付・期間境界、IP欠落、再実行の二重加算、表示しきい値変更、HTMLエスケープ。
