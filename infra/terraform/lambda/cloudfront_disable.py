"""Circuit breaker: disable the CloudFront distribution when the bandwidth
alarm fires, to stop a runaway egress bill.

Subscribed to the cost-guard SNS topic. CloudWatch publishes both ALARM and OK
notifications there (OK so the email says it recovered), so we only act on the
ALARM state and ignore everything else. Re-enabling the distribution is a
deliberate manual step (console / CLI); Terraform ignores `enabled` so an apply
won't silently undo the trip.
"""

import json
import os

import boto3


def _alarm_state(event):
    """Pull NewStateValue out of the SNS-wrapped CloudWatch notification.
    Returns None for a direct/manual invoke (treated as 'act')."""
    records = event.get("Records") or []
    for r in records:
        msg = r.get("Sns", {}).get("Message", "")
        try:
            return json.loads(msg).get("NewStateValue")
        except (ValueError, TypeError):
            return None
    return None


def handler(event, context):
    state = _alarm_state(event)
    if state is not None and state != "ALARM":
        print(f"ignoring notification state={state}")
        return

    dist_id = os.environ["DISTRIBUTION_ID"]
    cf = boto3.client("cloudfront")
    cfg = cf.get_distribution_config(Id=dist_id)
    dc = cfg["DistributionConfig"]

    if not dc["Enabled"]:
        print(f"distribution {dist_id} already disabled; nothing to do")
        return

    dc["Enabled"] = False
    cf.update_distribution(Id=dist_id, IfMatch=cfg["ETag"], DistributionConfig=dc)
    print(f"DISABLED CloudFront distribution {dist_id} due to bandwidth alarm")
