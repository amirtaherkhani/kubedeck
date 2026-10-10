#!/usr/bin/env python3
"""Move the approved k6 namespaced resources; keep Helm ownership namespace."""
import json
import subprocess
import sys

SOURCE = "observability-tests"
TARGET = "platform-tests"
NAMESPACED = {
    ("ServiceAccount", "k6-operator-controller"),
    ("Role", "k6-operator-leader-election-role"),
    ("RoleBinding", "k6-operator-leader-election-rolebinding"),
    ("Service", "k6-operator-controller-manager-metrics-service"),
    ("Deployment", "k6-operator-controller-manager"),
    ("ServiceMonitor", "controller-manager-metrics-monitor"),
}
BINDINGS = {"k6-operator-manager-rolebinding", "k6-operator-metrics-auth-rolebinding"}

def render(objects):
    seen = set()
    for obj in objects:
        meta = obj.get("metadata") or {}
        key = (obj.get("kind"), meta.get("name"))
        if key in NAMESPACED:
            if key in seen or meta.get("namespace") != SOURCE:
                raise ValueError("unexpected namespaced resource")
            seen.add(key)
            meta["namespace"] = TARGET
            if key[0] == "RoleBinding":
                move_subject(obj)
            if key[0] == "ServiceMonitor":
                selector = obj["spec"]["namespaceSelector"]
                if selector.get("matchNames") != [SOURCE]:
                    raise ValueError("unexpected metrics namespace selector")
                selector["matchNames"] = [TARGET]
        elif key[0] == "ClusterRoleBinding" and key[1] in BINDINGS:
            if key in seen:
                raise ValueError("duplicate binding")
            seen.add(key)
            move_subject(obj)
        elif meta.get("namespace") == SOURCE:
            raise ValueError("unreviewed source namespace resource")
    if seen != NAMESPACED | {("ClusterRoleBinding", name) for name in BINDINGS}:
        raise ValueError("missing approved resource")
    return objects

def move_subject(obj):
    expected = [{"kind": "ServiceAccount", "name": "k6-operator-controller", "namespace": SOURCE}]
    if obj.get("subjects") != expected:
        raise ValueError("unexpected RBAC subjects")
    obj["subjects"][0]["namespace"] = TARGET

def main():
    converted = subprocess.run(["yq", "-o=json", "-I=0", "."], input=sys.stdin.buffer.read(), capture_output=True, check=True)
    raw = converted.stdout.decode()
    decoder = json.JSONDecoder()
    objects = []
    while raw.strip():
        obj, end = decoder.raw_decode(raw.lstrip())
        raw = raw.lstrip()[end:]
        if isinstance(obj, dict):
            objects.append(obj)
    result = render(objects)
    sys.stdout.write("\n---\n".join(json.dumps(obj) for obj in result) + "\n")

if __name__ == "__main__":
    try:
        main()
    except (ValueError, KeyError, subprocess.CalledProcessError):
        sys.exit("k6 migration render rejected: input differs from reviewed scope")
