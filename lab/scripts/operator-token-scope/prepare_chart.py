#!/usr/bin/env python3
"""Build a checksum-pinned local chart. Offline only: never runs Helm/kubectl/network."""
import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import tarfile

DIGEST = "1dd2419b46f3b44d423a67bef2408a787e7f1b90598753d5730a46983bf0a82c"
TEMPLATE = "secrets-operator/templates/manager-rbac.yaml"
TOKEN_RULE = '''- apiGroups:
  - ""
  resources:
  - serviceaccounts/token
  verbs:
  - create
'''
NARROW_RULE = '''{{- if . }}
- apiGroups:
  - ""
  resources:
  - serviceaccounts/token
  resourceNames:
  {{- toYaml . | nindent 2 }}
  verbs:
  - create
{{- end }}
'''
GUARD = '''
{{- if not $isScopedMode }}{{ fail "KuchDesk token policy requires scopedRBAC and scopedNamespaces" }}{{ end }}
{{- if ne .Release.Namespace .Values.kuchdeskOperatorNamespace }}{{ fail "KuchDesk operator namespace mismatch" }}{{ end }}
{{- if ne (include "secrets-operator.serviceAccountName" .) .Values.kuchdeskOperatorServiceAccount }}{{ fail "KuchDesk operator service-account mismatch" }}{{ end }}
{{- if ne (len (uniq $namespaces)) (len $namespaces) }}{{ fail "KuchDesk watched namespaces must be unique" }}{{ end }}
{{- $policy := .Values.kuchdeskTokenRequestAccounts }}
{{- if not (kindIs "map" $policy) }}{{ fail "KuchDesk token policy map required" }}{{ end }}
{{- if ne (len $policy) (len $namespaces) }}{{ fail "KuchDesk token policy must cover exactly the watched namespaces" }}{{ end }}
{{- range $ns := $namespaces }}
{{- if not (hasKey $policy $ns) }}{{ fail "KuchDesk token policy missing namespace" }}{{ end }}
{{- $accounts := get $policy $ns }}
{{- if not (kindIs "slice" $accounts) }}{{ fail "KuchDesk service-account allowlist must be an array" }}{{ end }}
{{- range $name := $accounts }}
{{- if not (kindIs "string" $name) }}{{ fail "KuchDesk service-account name must be a string" }}{{ end }}
{{- if or (gt (len $name) 253) (not (regexMatch "^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$" $name)) }}{{ fail "Invalid KuchDesk service-account name" }}{{ end }}
{{- end }}
{{- end }}
'''


def load_policy(path):
    def unique_object(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError("duplicate policy key")
            result[key] = value
        return result
    policy = json.loads(Path(path).read_text(), object_pairs_hook=unique_object)
    if not isinstance(policy, dict) or set(policy) != {"operatorNamespace", "operatorServiceAccount", "namespaceAccounts"}:
        raise ValueError("unexpected policy fields")
    namespaces = policy["namespaceAccounts"]
    if not isinstance(namespaces, dict) or not namespaces:
        raise ValueError("namespace policy required")
    def valid(value):
        return isinstance(value, str) and len(value) <= 253 and all(
            len(part) <= 63 and re.fullmatch(r"[a-z0-9](?:[-a-z0-9]*[a-z0-9])?", part)
            for part in value.split("."))
    if not valid(policy["operatorNamespace"]) or not valid(policy["operatorServiceAccount"]):
        raise ValueError("invalid operator subject")
    for namespace, accounts in namespaces.items():
        if not valid(namespace) or "." in namespace or not isinstance(accounts, list):
            raise ValueError("invalid namespace policy")
        if any(not valid(a) for a in accounts) or len(set(accounts)) != len(accounts):
            raise ValueError("invalid service-account allowlist")
    return policy


def patch_template(source):
    old_include = '{{- include "secrets-operator.managerRules" $ }}'
    anchor = '{{- $isScopedMode := and $namespaces .Values.scopedRBAC }}'
    for text in (TOKEN_RULE, old_include, anchor):
        if source.count(text) != 1:
            raise ValueError("upstream RBAC contract changed")
    return source.replace(anchor, anchor + GUARD).replace(TOKEN_RULE, NARROW_RULE).replace(
        old_include, '{{- include "secrets-operator.managerRules" (get $.Values.kuchdeskTokenRequestAccounts $ns) }}')


def prepare(archive, output, policy_path):
    policy = load_policy(policy_path)
    archive = Path(archive)
    if archive.stat().st_size > 20 * 1024 * 1024:
        raise ValueError("chart archive too large")
    if hashlib.sha256(archive.read_bytes()).hexdigest() != DIGEST:
        raise ValueError("official chart checksum mismatch")
    contents = {}
    total = 0
    with tarfile.open(archive, "r:gz") as chart:
        for member in chart.getmembers():
            path = PurePosixPath(member.name)
            if path.is_absolute() or ".." in path.parts or not path.parts or path.parts[0] != "secrets-operator":
                raise ValueError("unsafe chart path")
            if member.isdir():
                continue
            if not member.isfile() or member.size > 20 * 1024 * 1024:
                raise ValueError("unsupported chart member")
            total += member.size
            if total > 40 * 1024 * 1024 or member.name in contents:
                raise ValueError("invalid chart archive")
            contents[member.name] = chart.extractfile(member).read()
    contents[TEMPLATE] = patch_template(contents[TEMPLATE].decode()).encode()
    values = "secrets-operator/values.yaml"
    if b"kuchdeskTokenRequestAccounts" in contents[values]:
        raise ValueError("unexpected existing token policy")
    overlay = {"kuchdeskTokenRequestAccounts": policy["namespaceAccounts"],
               "kuchdeskOperatorNamespace": policy["operatorNamespace"],
               "kuchdeskOperatorServiceAccount": policy["operatorServiceAccount"]}
    contents[values] += b"\n# KuchDesk explicit TokenRequest policy; do not use the unpatched upstream chart.\n"
    for key, value in overlay.items():
        contents[values] += (key + ": " + json.dumps(value, sort_keys=True) + "\n").encode()
    output = Path(output)
    # Never overwrite a previous chart, generated directory, or user file.
    output.mkdir(mode=0o700, parents=False, exist_ok=False)
    for name, data in contents.items():
        target = output / name
        target.parent.mkdir(parents=True, exist_ok=True)
        with target.open("xb") as file:
            file.write(data)
    # Explicit overlay is required with Helm --reuse-values, which can omit new defaults.
    with (output / "token-request-values.json").open("x") as file:
        json.dump(overlay, file, indent=2)
        file.write("\n")
    return output / "secrets-operator"


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--archive", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--policy", default=str(Path(__file__).with_name("policy.json")))
    args = parser.parse_args()
    try:
        print(prepare(args.archive, args.output, args.policy))
    except (ValueError, KeyError, OSError, tarfile.TarError) as error:
        parser.exit(1, f"chart preparation rejected: {error}\n")
