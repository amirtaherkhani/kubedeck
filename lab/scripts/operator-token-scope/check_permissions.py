#!/usr/bin/env python3
"""Read-only authorization reviews. Never requests a token or changes RBAC."""
import argparse
import json
from pathlib import Path
import subprocess
from prepare_chart import load_policy


def request(policy, namespace, name):
    op_ns = policy["operatorNamespace"]
    return {
        "apiVersion": "authorization.k8s.io/v1", "kind": "SubjectAccessReview",
        "spec": {
            "user": f"system:serviceaccount:{op_ns}:{policy['operatorServiceAccount']}",
            "groups": ["system:serviceaccounts", "system:serviceaccounts:" + op_ns, "system:authenticated"],
            "resourceAttributes": {
                "namespace": namespace, "verb": "create", "group": "",
                "resource": "serviceaccounts", "subresource": "token", "name": name,
            },
        },
    }


def kubectl_review(body):
    result = subprocess.run([
        "kubectl", "create", "--raw", "/apis/authorization.k8s.io/v1/subjectaccessreviews", "-f", "-",
    ], input=json.dumps(body), text=True, capture_output=True, timeout=15)
    if result.returncode:
        raise RuntimeError("authorization review unavailable")
    response = json.loads(result.stdout)
    status = response.get("status", {})
    if status.get("evaluationError") or not isinstance(status.get("allowed"), bool):
        raise RuntimeError("authorization review incomplete")
    return status["allowed"]


def check(policy, review=kubectl_review):
    reports = []
    for namespace, accounts in policy["namespaceAccounts"].items():
        denied_name = "kubedesk-unapproved-probe"
        while denied_name in accounts:
            denied_name += "x"
        for name, expected in [(a, True) for a in accounts] + [(denied_name, False)]:
            allowed = review(request(policy, namespace, name))
            reports.append({"namespace": namespace, "serviceAccount": name,
                            "allowed": allowed, "expectedAllowed": expected, "passed": allowed == expected})
    return reports


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--policy", default=str(Path(__file__).with_name("policy.json")))
    args = parser.parse_args()
    try:
        reports = check(load_policy(args.policy))
        print(json.dumps(reports, indent=2))
        raise SystemExit(0 if all(r["passed"] for r in reports) else 1)
    except (ValueError, RuntimeError, OSError, subprocess.TimeoutExpired):
        parser.exit(2, "authorization review unavailable or invalid; no permission conclusion\n")
