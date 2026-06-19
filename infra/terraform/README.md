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

### Route53でドメインを取得した場合（推奨・最小手間）

AWS をレジストラ兼DNSにすると、ネームサーバー変更も不要で全部自動になります。

1. Route53 コンソール → **Registered domains → Register domain** でドメイン購入
   （購入と同時に Hosted Zone が自動作成される）
2. `terraform.tfvars` に1行追加するだけ：
   ```hcl
   domain_name = "example.com"
   ```
3. `terraform apply`

これで Terraform が自動的に：
- ACM 証明書を **ALBリージョン** と **us-east-1（CloudFront用）** の両方で発行
- 証明書を **DNS検証**（Route53にCNAMEを自動追加）
- Route53 **Aliasレコード**を作成
  - `example.com` / `www.example.com` → ALB（フロント + `/api/*`）
  - `media.example.com` → CloudFront（HLS配信）
- ALB に HTTPS(443) リスナーを追加、CloudFront に独自ドメイン+証明書を適用

`acm.tf` / `route53.tf` が担当。証明書のDNS検証で数分かかることがあります。

### 他社で取得したドメイン（お名前.com 等）を使う場合

- **Route53にDNSを委任**: Route53でHosted Zoneを作り、出力された4つのNSをお名前.com側のネームサーバーに設定 → あとは上と同じく `domain_name` を入れて `apply`
- **自前の証明書を使う**: 既にACM証明書がある場合は `acm_certificate_arn_alb`（ALBリージョン）と `acm_certificate_arn_cloudfront`（us-east-1）をtfvarsに設定すると、自動発行をスキップして既存証明書を使います

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

## GitHub Actions でデプロイ（CI/CD + Change Tracking）

`.github/workflows/deploy.yml` が `master` への push（または手動実行）で動き、`push-images.sh` で ECR に push → 上記 EventBridge→Lambda が ECS を更新します。push 後に **New Relic Change Tracking**（デプロイマーカー）を NerdGraph 経由で記録します。

### 仕組み
```
git push (master)
   └─▶ GitHub Actions (OIDCでAWSへ)
          ├─ ./scripts/push-images.sh        … ECRへ build & push (:latest + git short sha)
          │     └─▶ EventBridge → Lambda → ECS rolling update（既存の仕組み）
          └─ ./scripts/newrelic-change-tracking.sh
                └─▶ NerdGraph changeTrackingCreateDeployment（version = git short sha）
```
- AWS 認証は **OIDC**（長期キーをGitHubに置かない）。`github_oidc.tf` が `<prefix>-gha-deploy` ロールを作成（権限は ECR push のみ）
- Change Tracking は `NEW_RELIC_USER_API_KEY` が未設定なら自動スキップ（`continue-on-error`なのでデプロイは止まらない）

### 一度だけの設定
1. `terraform apply`（ECR と OIDC ロールを作成）後、ロールARNを取得：
   ```bash
   terraform output github_actions_role_arn
   ```
2. GitHub リポジトリに登録：

   | 種類 | 名前 | 値 |
   |---|---|---|
   | Secret | `AWS_DEPLOY_ROLE_ARN` | 上記 `github_actions_role_arn` |
   | Secret | `NEW_RELIC_USER_API_KEY` | `NRAK-...`（User key。任意。Change Tracking用） |
   | Variable | `NEW_RELIC_APP_NAME` | NRのエンティティ名（dev例: `PerfectCatStreaming (dev)`） |
   | Variable | `NEW_RELIC_REGION` | `US` または `EU`（任意、既定US） |
   | Secret/Var | `VITE_NEW_RELIC_*` 各種 | フロントのブラウザ計装を焼き込む場合（任意） |

3. `git push` で自動デプロイ開始。GitHubの **Settings → Secrets and variables → Actions** で登録します。

> アカウントに既に GitHub OIDC プロバイダがある場合は、`create_github_oidc_provider = false` を `terraform.tfvars` に設定してください（重複作成エラー回避）。

### ローカルから手動デプロイ（従来どおり）
GitHub Actions を使わず手元から流すことも可能です：
```bash
./scripts/push-images.sh --env dev --region ap-northeast-1
# Change Tracking も手元で記録するなら：
NEW_RELIC_USER_API_KEY=NRAK-... NEW_RELIC_APP_NAME="PerfectCatStreaming (dev)" \
  VERSION="$(git rev-parse --short HEAD)" ./scripts/newrelic-change-tracking.sh
```

## RDS について

- MySQL 8.0 (`db.t4g.micro`、`gp3 20GB`、暗号化、private subnet)
- 起動時に backend が `videos` テーブルを `CREATE TABLE IF NOT EXISTS` で作る（マイグレーションツール不要のシンプル方式、tagsは `JSON` カラム）
- パスワードは `random_password` で生成して **Secrets Manager** に保存。ECS task は `secrets` 経由で `DATABASE_URL` を注入されます（exec role に GetSecretValue 権限）
- 接続: ECSタスクSGのみ許可、TLS（`tls=skip-verify`）

ローカル開発はベースの `docker-compose.yml` に `db: mysql:8.4` サービスがあるので `make up` でMySQLも自動起動します。`DATABASE_URL` を未設定で起動すれば in-memory + JSON にフォールバックします（旧モード維持）。

## New Relic APM について

backend は **HTTP / MySQL / S3 / バックグラウンドffmpeg** を自動計装します（`internal/observability/newrelic.go` + 各層）。

| 計装ポイント | 仕組み |
|---|---|
| HTTPトランザクション | chi route pattern (`GET /api/videos/{id}` 等) を txn名 に。`internal/http/middleware.go` の `NewRelicTxn` |
| MySQL クエリ | `nrmysql` ドライバ。リクエスト由来の ctx で `ExecContext`/`QueryContext` を呼ぶと自動でDB segment |
| S3 SDK 呼び出し | `nrawssdk-v2.AppendMiddlewares` で external segment |
| バックグラウンドffmpeg | `transcoder.Queue` がジョブごとに `app.StartTransaction("transcoder.job")` を発行、`ffmpeg.transcode` / `ffprobe.duration` を custom segment |

**ドメインカスタム属性**（NRQL で `FROM Transaction SELECT ... WHERE cat.breed = 'bengal'` のように絞り込めます）:

| 属性 | 付与トランザクション | 意味 |
|---|---|---|
| `cat.breed` / `cat.name` | upload / transcoder.job / GET video | 猫種・名前で性能を分解 |
| `video.id` / `video.status` | upload / GET・DELETE video / job | 対象動画・状態 |
| `video.count` | GET /api/videos | 一覧件数 |
| `upload.size_bytes` / `upload.content_type` | upload | アップロードサイズ・MIME |
| `transcode.publish_target` | transcoder.job | `s3` / `local` |
| `transcode.input_bytes` / `transcode.output_bytes` | transcoder.job | 入出力バイト数 |
| `transcode.video_duration_sec` / `transcode.wall_sec` | transcoder.job | 動画長・変換実時間 |
| `transcode.segment_count` | transcoder.job | 生成HLSセグメント数 |
| `transcode.realtime_factor` | transcoder.job | 変換実時間÷動画長（<1で実時間より高速＝効率指標） |

フロントの Kanpachi QoE イベント（`kanpachi.first_pounce` 等）にも `videoId` / `catName` / `breed` / `sessionId` 等が付与されます。

**有効化（Backend / APM）:**
```bash
# .env か CI で:
export TF_VAR_new_relic_license_key="<ingest license key>"
terraform apply
```
- `var.new_relic_license_key` が空のときは Secrets Manager リソース自体が作られず、ECS は NR 環境変数なしで起動 → アプリ側は `[newrelic] disabled (no license key)` で no-op
- セットすると Secrets Manager に保管され、ECS exec role に GetSecretValue 権限を付与、task に `NEW_RELIC_LICENSE_KEY` を注入

**Frontend (Browser Agent + Video Agent) のビルド時配線:**

フロントは Vite で **ビルド時に** env を埋め込みます。Browser Agent と Video Agent (HTML5 tracker + IMA Ads scaffold) は、5つの env が全部揃った時だけ初期化されます。

```bash
./scripts/push-images.sh \
  --env dev \
  --region ap-northeast-1 \
  -y \
  VITE_NEW_RELIC_LICENSE_KEY=... \
  VITE_NEW_RELIC_APP_ID=... \
  VITE_NEW_RELIC_ACCOUNT_ID=... \
  VITE_NEW_RELIC_TRUST_KEY=... \
  VITE_NEW_RELIC_AGENT_ID=...
```

（New Relic UI の Browser application > Snippet ページからコピペ。スクリプトは env を `--build-arg` 経由で Vite に渡します。）

実行時には何も注入されない（ビルド時に焼き込み済み）ので、ECS task の env は backend 用のみ。

## 既知の制約

- バックエンドは1レプリカ前提でも動きますが、RDS MySQL移行後は複数レプリカでも安全（メタデータが共有DBになったので）。スケールしたければ `backend_desired_count` を増やすだけ。
- 単一ビットレートHLS。マルチビットレート（ABR）化は後フェーズ。
- 既存のローカル `metadata.json` は RDS には自動同期されません（HLSファイルは `scripts/migrate-to-s3.sh` でS3へ）。必要なら起動後に再アップロードか、`mysql` クライアントで個別に INSERT。

## 撤去

```bash
terraform destroy
```

S3バケット (`media-XXXX`) は `force_destroy = true` にしてあるので中身ごと削除されます（dev向け設定）。本番では `false` に。
