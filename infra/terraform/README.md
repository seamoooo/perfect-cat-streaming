# perfect-cat-streaming — Terraform

ECS Fargate × CloudFront × S3 で `Bincho` と `Kanpachi` を世界に届けるためのインフラ定義。

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
   │ (nginx)│         │ (Go+ffm)│
   └────────┘         └─────────┘
   ECS Fargate         ECS Fargate
   (private subnet)    (private subnet)
```

- フロント (`/`) は nginx タスク
- バックエンド (`/api/*`, `/healthz`, `/meow`) は Go タスク
- 動画HLSは S3 (private) → CloudFront (OAC) で配信
- CloudFrontのレスポンスヘッダーポリシーで CORS 付与
- プレイリスト `*.m3u8` は短TTL（5s）、セグメントは `Managed-CachingOptimized`

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

イメージをビルドして push（リポジトリのルートから）：

```bash
ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
REGION=ap-northeast-1
aws ecr get-login-password --region $REGION | \
  docker login --username AWS --password-stdin $ACCOUNT.dkr.ecr.$REGION.amazonaws.com

# backend
docker build -t perfect-cat-backend:latest --target runtime ./backend
docker tag perfect-cat-backend:latest $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/perfect-cat-dev-backend:latest
docker push $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/perfect-cat-dev-backend:latest

# frontend (本番ビルド: CloudFrontのドメインを VITE_API_BASE_URL に渡しても良いが、
# ALB同居のためデフォルトの相対URL運用が楽)
docker build \
  --build-arg VITE_API_BASE_URL="" \
  -t perfect-cat-frontend:latest ./frontend
docker tag perfect-cat-frontend:latest $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/perfect-cat-dev-frontend:latest
docker push $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/perfect-cat-dev-frontend:latest
```

`terraform.tfvars` の `backend_image` / `frontend_image` を上の URL に書き換えて、本番applyへ：

```bash
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

## 既知の制約（MVP）

- メタデータ（`Video` レコード）は backend コンテナ内 `metadata.json` に保存しているため、ECS タスクの差し替えで消えます。再アップロードは可能。本番運用時は DynamoDB か RDS への移行を推奨。
- バックエンドは1レプリカ前提（メタデータをタスク間で共有しないため）。
- 単一ビットレートHLS。マルチビットレート（ABR）化は後フェーズ。

## 撤去

```bash
terraform destroy
```

S3バケット (`media-XXXX`) は `force_destroy = true` にしてあるので中身ごと削除されます（dev向け設定）。本番では `false` に。
