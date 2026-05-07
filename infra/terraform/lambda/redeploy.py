"""
Redeploy ECS services on ECR push.

Triggered by EventBridge whenever a watched ECR repo receives a PUSH for the
configured tag. Maps the repository name to a service name (env REPO_TO_SERVICE,
formatted as "repoA=svcA,repoB=svcB") and calls UpdateService with
forceNewDeployment=True so the task picks up the freshly-pushed image.
"""

import json
import logging
import os

import boto3

logger = logging.getLogger()
logger.setLevel(logging.INFO)

ecs = boto3.client("ecs")


def _parse_mapping(s: str) -> dict[str, str]:
    out: dict[str, str] = {}
    for pair in s.split(","):
        pair = pair.strip()
        if not pair:
            continue
        if "=" not in pair:
            continue
        k, v = pair.split("=", 1)
        out[k.strip()] = v.strip()
    return out


def handler(event, _context):
    logger.info("event: %s", json.dumps(event))
    cluster = os.environ["CLUSTER"]
    mapping = _parse_mapping(os.environ.get("REPO_TO_SERVICE", ""))

    detail = event.get("detail", {})
    repo = detail.get("repository-name") or ""
    tag = detail.get("image-tag") or ""

    service = mapping.get(repo)
    if not service:
        logger.info("no mapping for repo=%s, skipping", repo)
        return {"skipped": True, "repo": repo}

    logger.info("redeploying repo=%s tag=%s -> cluster=%s service=%s", repo, tag, cluster, service)
    resp = ecs.update_service(cluster=cluster, service=service, forceNewDeployment=True)
    return {
        "redeployed": True,
        "repo": repo,
        "service": service,
        "deployment": resp.get("service", {}).get("deployments", [{}])[0].get("id"),
    }
