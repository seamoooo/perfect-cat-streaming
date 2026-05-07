# perfect-cat-streaming — Terraform

ECS Fargate × CloudFront × S3 × RDS MySQL で `Bincho` と `Kanpachi` を世界に届けるためのインフラ定義。

## 何が立つか

```
                       ┌──────────── CloudFront ─────────────┐
                       │  cors policy / playlist short-TTL    │
         Browser ──┐   │  *.ts immutable cache                │
                   │   └──────────────────┬───────────────────┘
                   │                      │ (OAC sigv4)
                   ▼                      ▼
              ┌─────────┐          ┌─────────────┐
              │   ALB   │          │ S3 (private) │
              └────┬────┘          │  hls/<id>/   │
                   │               └─────────────┘
         ┌─────────┴─────────┐            ▲
         │                   │            │ PutObject
    ┌────────┐         ┌─────────┐        │ via task IAM role
    │frontend│         │ backend │────────┘
    │ (nginx)│         │ (Go+ffm)│───┐
    └────────┘         └─────────┘   │ DATABASE_URL (Secrets Manager)
    ECS Fargate         ECS Fargate  ▼
    (private)           (private)  ┌──────────────┐
                                   │ RDS MySQL │
                                   │  (private)   │
                                   └──────────────┘

  ECR push (image-tag = "latest") ─▶ EventBridge ─▶ Lambda
                                                      └─▶ ecs:UpdateService(forceNewDeployment=true)
```

- フロント (`/`) は nginx タスク
- バックエンド (`/api/*`, `/healthz`, `/meow`) は Go タスク
- 動画HLSは S3 (private) → CloudFront (OAC) で配信
- CloudFrontのレスポンスヘッダーポリシーで CORS 付与
- プレイリスト `*.m3u8` は短TTL（5s）、セグメントは `Managed-CachingOptimized`
- メタデータは RDS MySQL、DSN は Secrets Manager 経由で task に注入
- ECRに新しいイメージが push されると EventBridge Rule → Lambda → ECS の force-new-deployment が走る

## 前提

- AWS CLI が認証済み (`aws sts get-caller-identity` が通る)
- Terraform >= 1.5
- Docker（イメージビルド/プッシュ用）

## 初回デプロイ手順

```bash
cd infra/terraform
cp terraform.tfvars.example terraform.tfvars     # backend_image/frontend_image は仮値のままでOK
terraform init
terraform apply -target=aws_ecr_repository.backend -target=aws_ecr_repository.frontend
```

ECR のリポジトリだけ先に作って URL を控えます：

```bash
terraform output ecr_backend_url
terraform output ecr_frontend_url
```

イメージビルド & push は `scripts/push-images.sh` を使えば1コマンドです：

```bash
cd ../..   # repo root
./scripts/push-images.sh --env dev --region ap-northeast-1
```

このスクリプトは：
- `aws ecr get-login-password` で自動ログイン
- backend (`runtime` ステージ) と frontend を `linux/amd64` でビルド
- `:latest` と git short SHA の2タグで push

push 後、ECRイベントを EventBridge が拾い、Lambda が `ecs:UpdateService --force-new-deployment` を呼ぶので、ECSが自動で新イメージに切り替わります。

`terraform.tfvars` の `backend_image` / `frontend_image` を ECR の URL に書き換えて、本番applyへ：

```bash
cd infra/terraform
# backend_image  = "<acct>.dkr.ecr.ap-northeast-1.amazonaws.com/perfect-cat-dev-backend:latest"
# frontend_image = "<acct>.dkr.ecr.ap-northeast-1.amazonaws.com/perfect-cat-dev-frontend:latest"
terraform apply
```

完了後の出力を確認：

```bash
terraform output frontend_url             # ブラウザで開く
terraform output cloudfront_domain        # backendのMEDIA_BASE_URLに自動セット済
terraform output media_bucket             # S3バケット名（移行スクリプトで使う）
```

## 既存のローカル動画をS3に移行

```bash
cd ../..   # repo root
./scripts/migrate-to-s3.sh \
  --bucket "$(cd infra/terraform && terraform output -raw media_bucket)" \
  --region ap-northeast-1
```

詳細は `scripts/migrate-to-s3.sh --help`。

## カスタムドメインで HTTPS にしたい場合

1. `domain_name = "cats.example.com"` を `terraform.tfvars` に追加
2. ACM cert を **ALB と同リージョン** で発行 → `acm_certificate_arn_alb`
3. ACM cert を **us-east-1** で発行 → `acm_certificate_arn_cloudfront`
4. `terraform apply`
5. Route53で
   - `cats.example.com` → ALB
   - `media.cats.example.com` → CloudFront

## 自動再デプロイの仕組み

1. `./scripts/push-images.sh` で ECR に `:latest` タグで push
2. EventBridge Rule `<prefix>-ecr-push` が `ECR Image Action / PUSH / SUCCESS / image-tag=latest` を捕捉
3. Lambda `<prefix>-redeploy` が `repository-name → service-name` のマッピングで対象 service を特定
4. `ecs:UpdateService(forceNewDeployment=True)` を呼び、ECS が新イメージで rolling update
5. `deployment_circuit_breaker` が有効なので、新タスクが unhealthy ならロールバック

ロールアウト状況は：
```bash
aws ecs describe-services \
  --cluster perfect-cat-dev-cluster \
  --services perfect-cat-dev-backend perfect-cat-dev-frontend \
  --query 'services[].deployments[0]' --output table
```

## RDS について

- MySQL 8.0 (`db.t4g.micro`、`gp3 20GB`、暗号化、private subnet)
- 起動時に backend が `videos` テーブルを `CREATE TABLE IF NOT EXISTS` で作る（マイグレーションツール不要のシンプル方式、tagsは `JSON` カラム）
- パスワードは `random_password` で生成して **Secrets Manager** に保存。ECS task は `secrets` 経由で `DATABASE_URL` を注入されます（exec role に GetSecretValue 権限）
- 接続: ECSタスクSGのみ許可、TLS（`tls=skip-verify`）

ローカル開発はベースの `docker-compose.yml` に `db: mysql:8.4` サービスがあるので `make up` でMySQLも自動起動します。`DATABASE_URL` を未設定で起動すれば in-memory + JSON にフォールバックします（旧モード維持）。

## 既知の制約

- バックエンドは1レプリカ前提でも動きますが、Postgres移行後は複数レプリカでも安全（メタデータが共有DBになったので）。スケールしたければ `backend_desired_count` を増やすだけ。
- 単一ビットレートHLS。マルチビットレート（ABR）化は後フェーズ。
- 既存のローカル `metadata.json` は RDS には自動同期されません（HLSファイルは `scripts/migrate-to-s3.sh` でS3へ）。必要なら起動後に再アップロードか、`psql` で個別に INSERT。

## 撤去

```bash
terraform destroy
```

S3バケット (`media-XXXX`) は `force_destroy = true` にしてあるので中身ごと削除されます（dev向け設定）。本番では `false` に。
