#!/usr/bin/env bash
# Build & push backend / frontend images to ECR. ECR push fires EventBridge →
# Lambda → ecs:UpdateService(forceNewDeployment=true), so the new image is
# picked up automatically by ECS.

set -euo pipefail

# --- defaults ---
SERVICE="all"      # all | backend | frontend
TAG="latest"
EXTRA_TAG=""       # additional immutable tag (default: short git sha if available)
PROJECT="perfect-cat"
ENV_NAME="dev"
REGION="${AWS_REGION:-${AWS_DEFAULT_REGION:-ap-northeast-1}}"
VITE_API_BASE_URL="${VITE_API_BASE_URL:-}"
VITE_NEW_RELIC_LICENSE_KEY="${VITE_NEW_RELIC_LICENSE_KEY:-}"
VITE_NEW_RELIC_APP_ID="${VITE_NEW_RELIC_APP_ID:-}"
ASSUME_YES=0

usage() {
  cat <<EOF
Build & push backend/frontend images to ECR (Bincho × Kanpachi edition).

Usage: $(basename "$0") [options]

Options:
  --service NAME      all | backend | frontend (default: all)
  --tag TAG           Tag to push as the auto-deploy trigger (default: latest)
  --extra-tag TAG     Additional immutable tag; default: short git sha
  --project NAME      Project name prefix in ECR (default: perfect-cat)
  --env NAME          Environment suffix (default: dev) — gives <project>-<env>-<service>
  --region REGION     AWS region (default: \$AWS_REGION or ap-northeast-1)
  --api-base URL      VITE_API_BASE_URL build arg for frontend (default: empty = same domain)
  -y, --yes           Skip confirmation prompt
  -h, --help          This help

Examples:
  $(basename "$0") --env dev
  $(basename "$0") --service backend --tag v1.2.3
  AWS_REGION=us-east-1 $(basename "$0") --env prod --api-base https://api.cats.example.com
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --service)   SERVICE="$2"; shift 2 ;;
    --tag)       TAG="$2"; shift 2 ;;
    --extra-tag) EXTRA_TAG="$2"; shift 2 ;;
    --project)   PROJECT="$2"; shift 2 ;;
    --env)       ENV_NAME="$2"; shift 2 ;;
    --region)    REGION="$2"; shift 2 ;;
    --api-base)  VITE_API_BASE_URL="$2"; shift 2 ;;
    -y|--yes)    ASSUME_YES=1; shift ;;
    -h|--help)   usage; exit 0 ;;
    *) echo "Unknown arg: $1" >&2; usage; exit 1 ;;
  esac
done

if [[ "$SERVICE" != "all" && "$SERVICE" != "backend" && "$SERVICE" != "frontend" ]]; then
  echo "ERROR: --service must be all|backend|frontend" >&2; exit 1
fi

for cmd in aws docker; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "ERROR: $cmd not in PATH" >&2; exit 1; }
done

cd "$(git rev-parse --show-toplevel 2>/dev/null || dirname "$0")/.."

ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
REGISTRY="$ACCOUNT.dkr.ecr.$REGION.amazonaws.com"
BACKEND_REPO="$PROJECT-$ENV_NAME-backend"
FRONTEND_REPO="$PROJECT-$ENV_NAME-frontend"

if [[ -z "$EXTRA_TAG" ]]; then
  EXTRA_TAG=$(git rev-parse --short HEAD 2>/dev/null || true)
fi

echo "================================================================"
echo " perfect-cat-streaming → ECR push"
echo "================================================================"
echo " Account:   $ACCOUNT"
echo " Region:    $REGION"
echo " Registry:  $REGISTRY"
echo " Service:   $SERVICE"
echo " Tag:       $TAG${EXTRA_TAG:+, $EXTRA_TAG}"
[[ "$SERVICE" != "backend" ]] && echo " API base:  ${VITE_API_BASE_URL:-(empty = same-domain ALB)}"
echo "================================================================"

if [[ $ASSUME_YES -ne 1 ]]; then
  read -r -p "Proceed? [y/N] " ans
  case "$ans" in y|Y|yes|YES) ;; *) echo "Aborted."; exit 0 ;; esac
fi

echo "[+] Logging in to ECR ..."
aws ecr get-login-password --region "$REGION" | \
  docker login --username AWS --password-stdin "$REGISTRY"

push_one() {
  local repo="$1" image="$2"
  docker tag "$image:$TAG" "$REGISTRY/$repo:$TAG"
  docker push "$REGISTRY/$repo:$TAG"
  if [[ -n "$EXTRA_TAG" ]]; then
    docker tag "$image:$TAG" "$REGISTRY/$repo:$EXTRA_TAG"
    docker push "$REGISTRY/$repo:$EXTRA_TAG"
  fi
}

if [[ "$SERVICE" == "all" || "$SERVICE" == "backend" ]]; then
  echo
  echo "[backend] building runtime image ..."
  docker build --platform linux/amd64 --target runtime \
    -t "perfect-cat-backend:$TAG" \
    ./backend
  echo "[backend] pushing ..."
  push_one "$BACKEND_REPO" "perfect-cat-backend"
fi

if [[ "$SERVICE" == "all" || "$SERVICE" == "frontend" ]]; then
  echo
  echo "[frontend] building production image ..."
  docker build --platform linux/amd64 \
    --build-arg VITE_API_BASE_URL="$VITE_API_BASE_URL" \
    --build-arg VITE_NEW_RELIC_LICENSE_KEY="$VITE_NEW_RELIC_LICENSE_KEY" \
    --build-arg VITE_NEW_RELIC_APP_ID="$VITE_NEW_RELIC_APP_ID" \
    -t "perfect-cat-frontend:$TAG" \
    ./frontend
  echo "[frontend] pushing ..."
  push_one "$FRONTEND_REPO" "perfect-cat-frontend"
fi

echo
echo "Done. 🐾"
echo
echo "EventBridge → redeploy Lambda → ecs:UpdateService should have fired."
echo "Watch the rollout:"
echo "  aws ecs describe-services --cluster $PROJECT-$ENV_NAME-cluster \\"
echo "    --services $PROJECT-$ENV_NAME-backend $PROJECT-$ENV_NAME-frontend \\"
echo "    --query 'services[].deployments[0].{svc:serviceName,desired:desiredCount,running:runningCount,status:rolloutState,reason:rolloutStateReason}' \\"
echo "    --output table --region $REGION"
