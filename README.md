# perfect-cat-streaming

> Bincho（シャム）と Kanpachi（ベンガル）のための、完璧なHLSストリーミング配信サイト 🐾

```
        /\_/\        /\_/\
       ( o.o )      ( ^.^ )
        > ^ <        > ^ <
        Bincho       Kanpachi
        (siamese)    (bengal)
```

MP4をアップロードするとバックグラウンドでHLS（`.m3u8` / `.ts`）に変換し、ギャラリー一覧と詳細画面で再生できます。動画プレイヤーはQoEメトリクス（TTFF / リバッファ / エラー / ビットレート遷移）を捕捉して任意のテレメトリ先（New Relic等）に送信します。

## 構成

- **Backend (`backend/`)**: Go (`net/http` + `chi`) + `ffmpeg`同梱コンテナ
- **Frontend (`frontend/`)**: React + Vite + TypeScript + `hls.js`
- **Orchestration**: Docker Compose（本番ベース + ローカル開発override）

## クイックスタート

```bash
make up        # = docker compose up --build (env未作成なら .env も生成)
# or
make pounce    # 同上（猫の愛バージョン）
```

- フロント: http://localhost:8000 （本番ビルド配信）または http://localhost:5173（dev）
- API: http://localhost:8080
- ヘルスチェック: http://localhost:8080/healthz
- おまけ: http://localhost:8080/meow （Bincho/Kanpachiの鳴き声がランダムで返る）

停止: `make down` / `make nap`

## ディレクトリ概要

```
backend/      Go API server (ffmpeg in container)
frontend/     React + Vite + hls.js
data/         host-mounted volume root
  uploads/    生のMP4
  hls/        変換後のHLSプレイリストとセグメント
```

## 環境変数（パス吸収）

ローカルとサーバーで動作環境が変わってもアプリ側コードは書き換え不要です。

| 変数 | デフォルト | 用途 |
|---|---|---|
| `STORAGE_UPLOAD_DIR` | `/var/app/data/uploads`（コンテナ内） | アップロード保存先 |
| `STORAGE_HLS_DIR` | `/var/app/data/hls` | HLS出力先 |
| `PUBLIC_BASE_URL` | `http://localhost:8080` | フロントから返すストリーミングURLのベース |
| `ALLOWED_ORIGINS` | `http://localhost:5173,http://localhost:8000` | CORS許可元 |
| `HOST_DATA_DIR` | `./data` | **ホスト側**ボリュームマウント先（本番では永続ボリューム/EBSへ） |
| `FFMPEG_BIN` | `ffmpeg` | ffmpegパス上書き用 |

コンテナ内パスは固定、ホスト側マウント先のみ `HOST_DATA_DIR` で切替する設計。

## QoE メトリクス

`<KanpachiPlayer>` がベンガル猫のように獲物（メトリクス）を狩ります。

- イベント名空間: `kanpachi.*`
  - `kanpachi.first_pounce` — TTFF（最初のフレーム到達時間）
  - `kanpachi.hairball_start` / `kanpachi.hairball_end` — リバッファ開始/終了
  - `kanpachi.hiss` — 再生エラー
  - `kanpachi.stretch` — ビットレート遷移
- カスタム属性: `videoId`, `catName`, `breed`, `title`, `tags`, `sessionId`, `playerVersion`, `userAgent`
- 送信先は `KanpachiSink` インタフェースで差し替え可能（dev: `ConsoleKanpachiSink` / prod: `NewRelicKanpachiSink`）

実装は `frontend/src/lib/telemetry/` と `frontend/src/hooks/useKanpachiMetrics.ts` を参照。

## 開発者デモ（カオス注入）

New Relic でのトラブルシュート練習用に、**動画の説明文（description）に英単語キーワードを 1 つ入れる**と、その層に意図的な障害を注入します。説明文は DB（`videos.description`）にそのまま保存され、各層が読み取って単語照合するだけ（SQL やシェルには渡さないので安全。クエリは全てプレースホルダでパラメータ化済み）。

| キーワード | 層 | 挙動 | New Relic 目印 |
|---|---|---|---|
| `SRE` | バックエンド | トランスコードのスループットが大幅悪化（`-re` + x264 `veryslow` で実時間変換に） | `chaos.injected = sre_slow_transcode` / `transcode.realtime_factor` が ~0.1 → ~1.0+ |
| `player` | フロント | HLS プレイヤーで再生エラー（`kanpachi.hiss` + NR Video Agent） | `reason = chaos` |
| `frontend` | フロント | 詳細画面のレンダリングで例外 → ErrorBoundary が捕捉 | `chaos.injected = frontend_render_error` |
| `backend` | バックエンド | 詳細取得 API `GET /api/videos/:id` が HTTP 500 を返す | `chaos.injected = backend_500` |

- 判定は **ASCII トークン単位の大文字小文字無視マッチ**。日本語の通常文には反応せず、`SRE` / `sre` どちらも有効。
- 基本は 1 キーワードのみ想定。複数入れた場合は `SRE → player → frontend → backend` の優先順位で 1 つに収束（決定的）。
- 判定ロジックは backend `internal/chaos/directive.go` と frontend `src/lib/chaos.ts` のミラー実装（同一テストケースで担保）。

## 開発

```bash
make up               # 全部立ち上げ（Vite dev + Go run）
make logs             # ログ追跡
make fmt              # フォーマット
make test             # 最小テスト

# 直接Goを叩きたい場合（コンテナ外）
cd backend && go run ./cmd/server
```

## API

| Method | Path | 用途 |
|---|---|---|
| `POST` | `/api/videos` | multipart で MP4 受領、即 202、変換は非同期 |
| `GET` | `/api/videos` | 一覧（status: `pending` / `processing` / `ready` / `error`） |
| `GET` | `/api/videos/:id` | 詳細（QoE属性に使うメタ含む） |
| `GET` | `/media/:id/index.m3u8` | HLSプレイリスト |
| `GET` | `/media/:id/{segment}.ts` | HLSセグメント |
| `GET` | `/healthz` | ヘルスチェック |
| `GET` | `/meow` | Bincho/Kanpachiの鳴き声（30+バリエーション） |

## デプロイ

### シンプルな単一サーバー (Compose)

```bash
HOST_DATA_DIR=/srv/perfect-cat/data \
PUBLIC_BASE_URL=https://api.example.com \
ALLOWED_ORIGINS=https://cats.example.com \
docker compose -f docker-compose.yml up -d --build
```

`docker-compose.override.yml` は本番では読み込ませません（`-f docker-compose.yml` のみで起動）。

### AWS (ECS Fargate × CloudFront × S3)

`infra/terraform/` に Terraform 定義一式があります。詳細は `infra/terraform/README.md`。

```
ECS Fargate (backend Go + ffmpeg, frontend nginx)
   ├─ behind ALB (path routing: /api/* → backend, / → frontend)
   └─ backend writes HLS to:
S3 (private) ── OAC ──▶ CloudFront ──▶ browser hls.js
```

backend の S3 publisher は `S3_BUCKET` env を立てた時だけ有効化されます（未設定ならローカルディスクからの `/media/*` 配信のまま）。Terraform の ECS task 定義で自動セットされます。

#### 既存のローカル動画を一気にS3へ移行

```bash
./scripts/migrate-to-s3.sh \
  --bucket "$(cd infra/terraform && terraform output -raw media_bucket)" \
  --region ap-northeast-1
```

`--include-uploads` で原本MP4も同期、`--dry-run` で安全確認、`-y` でプロンプトスキップ。
