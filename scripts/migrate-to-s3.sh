#!/usr/bin/env bash
# Bulk-upload locally-stored cat clips (HLS + originals) to S3.
#
# Reads HLS from $DATA_DIR/hls/<videoID>/{index.m3u8,seg_*.ts} and uploads to
#   s3://$BUCKET/$HLS_PREFIX/<videoID>/...
# Optionally also pushes raw uploads from $DATA_DIR/uploads/.
#
# Idempotent: aws s3 sync only uploads new/changed files.

set -euo pipefail

DATA_DIR="./data"
HLS_PREFIX="hls"
UPLOADS_PREFIX="uploads"
INCLUDE_UPLOADS=0
BUCKET=""
REGION=""
DRY_RUN=0
ASSUME_YES=0

usage() {
  cat <<EOF
Bulk-upload local cat clips to S3 (Bincho × Kanpachi edition).

Usage: $(basename "$0") --bucket NAME [options]

Required:
  --bucket NAME              Target S3 bucket (Terraform: media_bucket output)

Options:
  --region REGION            AWS region (default: from AWS env / config)
  --data-dir DIR             Local data dir (default: ./data)
  --hls-prefix PREFIX        S3 key prefix for HLS dirs (default: hls)
  --uploads-prefix PREFIX    S3 key prefix for raw uploads (default: uploads)
  --include-uploads          Also push original MP4s from \$DATA_DIR/uploads/
  --dry-run                  Show what would be uploaded; do not transfer
  -y, --yes                  Skip confirmation prompt
  -h, --help                 This help

Example:
  $(basename "$0") --bucket perfect-cat-dev-media-a1b2c3 --region ap-northeast-1 -y
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bucket)          BUCKET="$2"; shift 2 ;;
    --region)          REGION="$2"; shift 2 ;;
    --data-dir)        DATA_DIR="$2"; shift 2 ;;
    --hls-prefix)      HLS_PREFIX="$2"; shift 2 ;;
    --uploads-prefix)  UPLOADS_PREFIX="$2"; shift 2 ;;
    --include-uploads) INCLUDE_UPLOADS=1; shift ;;
    --dry-run)         DRY_RUN=1; shift ;;
    -y|--yes)          ASSUME_YES=1; shift ;;
    -h|--help)         usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 1 ;;
  esac
done

if [[ -z "$BUCKET" ]]; then
  echo "ERROR: --bucket is required" >&2
  usage
  exit 1
fi

if ! command -v aws >/dev/null 2>&1; then
  echo "ERROR: aws CLI not found in PATH" >&2
  exit 1
fi

HLS_LOCAL="$DATA_DIR/hls"
UPLOADS_LOCAL="$DATA_DIR/uploads"

if [[ ! -d "$HLS_LOCAL" ]]; then
  echo "ERROR: HLS directory not found: $HLS_LOCAL" >&2
  exit 1
fi

# Count clips for the prompt
CLIP_COUNT=$(find "$HLS_LOCAL" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')
HLS_BYTES=$(du -sk "$HLS_LOCAL" 2>/dev/null | awk '{print $1}')
HLS_HUMAN=$(awk -v k="$HLS_BYTES" 'BEGIN { printf "%.1f MB", k/1024 }')

echo "================================================================"
echo " perfect-cat-streaming → S3 migration"
echo "================================================================"
echo " Bucket          : s3://$BUCKET"
[[ -n "$REGION" ]] && echo " Region          : $REGION"
echo " Local HLS dir   : $HLS_LOCAL ($CLIP_COUNT clips, $HLS_HUMAN)"
echo " HLS prefix      : $HLS_PREFIX/"
if [[ $INCLUDE_UPLOADS -eq 1 ]]; then
  if [[ -d "$UPLOADS_LOCAL" ]]; then
    UPL_COUNT=$(find "$UPLOADS_LOCAL" -type f ! -name '.gitkeep' | wc -l | tr -d ' ')
    UPL_BYTES=$(du -sk "$UPLOADS_LOCAL" 2>/dev/null | awk '{print $1}')
    UPL_HUMAN=$(awk -v k="$UPL_BYTES" 'BEGIN { printf "%.1f MB", k/1024 }')
    echo " Uploads dir     : $UPLOADS_LOCAL ($UPL_COUNT files, $UPL_HUMAN)"
    echo " Uploads prefix  : $UPLOADS_PREFIX/"
  else
    echo " Uploads dir     : (not found, skipping)"
    INCLUDE_UPLOADS=0
  fi
fi
[[ $DRY_RUN -eq 1 ]] && echo " Mode            : DRY RUN (no upload)"
echo "================================================================"

if [[ $ASSUME_YES -ne 1 ]]; then
  read -r -p "Proceed? [y/N] " ans
  case "$ans" in
    y|Y|yes|YES) ;;
    *) echo "Aborted."; exit 0 ;;
  esac
fi

REGION_ARG=()
[[ -n "$REGION" ]] && REGION_ARG=(--region "$REGION")

DRY_ARG=()
[[ $DRY_RUN -eq 1 ]] && DRY_ARG=(--dryrun)

# --- Upload HLS ---
# Per-file content-type via two passes — sync once for .m3u8, once for .ts.
# (aws s3 sync doesn't support per-extension content-type in one shot.)

echo
echo "[1/2] Uploading playlists (.m3u8) ..."
aws s3 sync "$HLS_LOCAL/" "s3://$BUCKET/$HLS_PREFIX/" \
  --exclude "*" \
  --include "*.m3u8" \
  --content-type "application/vnd.apple.mpegurl" \
  --cache-control "public, max-age=10" \
  "${REGION_ARG[@]}" \
  "${DRY_ARG[@]}"

echo
echo "[2/2] Uploading segments (.ts) ..."
aws s3 sync "$HLS_LOCAL/" "s3://$BUCKET/$HLS_PREFIX/" \
  --exclude "*" \
  --include "*.ts" \
  --content-type "video/mp2t" \
  --cache-control "public, max-age=31536000, immutable" \
  "${REGION_ARG[@]}" \
  "${DRY_ARG[@]}"

if [[ $INCLUDE_UPLOADS -eq 1 ]]; then
  echo
  echo "[+] Uploading raw originals ..."
  aws s3 sync "$UPLOADS_LOCAL/" "s3://$BUCKET/$UPLOADS_PREFIX/" \
    --exclude ".gitkeep" \
    "${REGION_ARG[@]}" \
    "${DRY_ARG[@]}"
fi

echo
echo "Done. 🐾"
echo
echo "Verify with:"
echo "  aws s3 ls s3://$BUCKET/$HLS_PREFIX/ ${REGION:+--region $REGION}"
echo
echo "After this, the backend running in ECS (with S3_BUCKET env var) will"
echo "publish NEW clips automatically; this script handles the existing ones."
