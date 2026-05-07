# Schema (MySQL × sqldef)

`schema.sql` を **唯一の正** として、 [`mysqldef`](https://github.com/sqldef/sqldef)（schema-as-code ツール）で
ライブDBへ収束させます。手書きの ALTER は書きません。

## インストール

```bash
# macOS (Homebrew)
brew install sqldef/sqldef/mysqldef

# or: Go から (1.21+)
go install github.com/sqldef/sqldef/cmd/mysqldef@latest
```

## ローカル開発

`docker compose up` で `db` コンテナ (MySQL 8) が立ち上がります。ホスト 3306 に
ポートフォワード済みなので、Makefile 経由で叩けます。

```bash
make schema-dryrun       # diffだけ表示。何も変更しない
make schema-apply        # 差分を適用
make schema-export       # 現在のライブDBのDDLを出力
```

## RDS（AWS）

RDS は private subnet にあるため、踏み台 / SSM session manager / VPN 経由で
接続するか、一時的にローカルへ port-forward します。例えば `aws rds-data` を
通すか、SSM Session Manager の port forwarding で：

```bash
# 例: SSMトンネル経由
DATABASE_URL="$(aws secretsmanager get-secret-value \
   --secret-id perfect-cat-dev/database-url \
   --query SecretString --output text)"

# DSN を mysqldef 用に分解
HOST=...; PORT=3306; USER=...; PASS=...; DB=...

mysqldef --dry-run -h $HOST -P $PORT -u $USER -p $PASS $DB < infra/schema/schema.sql
mysqldef           -h $HOST -P $PORT -u $USER -p $PASS $DB < infra/schema/schema.sql
```

## ワークフロー

1. `infra/schema/schema.sql` を編集してコミット
2. CI/手元で `mysqldef --dry-run` を実行 → 期待通りの DDL か確認
3. `mysqldef` で適用
4. アプリのデプロイ

アプリ側の `repository/mysql.go` は起動時に `videos` テーブルの存在のみ確認
（fail-fast）。schema 自体は触りません。
