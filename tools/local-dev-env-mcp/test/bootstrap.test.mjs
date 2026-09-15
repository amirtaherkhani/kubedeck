import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const packageRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const repositoryRoot = join(packageRoot, "..", "..");
const bootstrap = join(repositoryRoot, "scripts", "local-dev-env-bootstrap.sh");

function registrationPath(metadata) {
  return execFileSync("/bin/bash", ["-c", 'source "$1"; registration_path_for_file "$2"', "test", bootstrap, metadata], {
    encoding: "utf8"
  }).trim();
}

test("discovers literal and templated chart scopes", () => {
  assert.equal(
    registrationPath(join(repositoryRoot, "apps", "platform", "storage", "templates", "infisicalsecrets.yaml")),
    "/apps/platform-storage"
  );
  assert.equal(
    registrationPath(join(repositoryRoot, "apps", "dev", "n8n", "templates", "infisicalsecret.yaml")),
    "/apps/development-tools/n8n"
  );
});

test("credential reconciliation revokes only superseded connector credentials", () => {
  const log = join(mkdtempSync(join(tmpdir(), "local-dev-env-bootstrap-")), "revocations.log");
  execFileSync("/bin/bash", ["-c", `
    source "$1"
    admin_api() {
      if [[ "$2" == "GET" ]]; then
        printf '%s\\n' '{"clientSecretData":[{"id":"current","description":"local-dev-env-mcp on this Mac","isClientSecretRevoked":false},{"id":"old","description":"local-dev-env-mcp on this Mac","isClientSecretRevoked":false},{"id":"revoked","description":"local-dev-env-mcp on this Mac","isClientSecretRevoked":true},{"id":"other","description":"another client","isClientSecretRevoked":false}]}'
      else
        printf '%s\\n' "$3" >>"$TEST_LOG"
      fi
    }
    revoke_superseded_client_secrets token identity current
  `, "test", bootstrap], { env: { ...process.env, TEST_LOG: log } });
  assert.equal(readFileSync(log, "utf8").trim(), "/auth/universal-auth/identities/identity/client-secrets/old/revoke");
});

test("credential replacement stores a seven-day secret before revoking old credentials", () => {
  const log = join(mkdtempSync(join(tmpdir(), "local-dev-env-bootstrap-")), "rotation.log");
  execFileSync("/bin/bash", ["-c", `
    source "$1"
    configure_universal_auth() { printf '%s\\n' '{"identityUniversalAuth":{"clientId":"new-client"}}'; }
    credentials_are_valid() { return 1; }
    admin_api() {
      printf 'create %s\\n' "$4" >>"$TEST_LOG"
      printf '%s\\n' '{"clientSecret":"new-secret","clientSecretData":{"id":"new-secret-id"}}'
    }
    keychain_put() { printf 'store %s\\n' "$1" >>"$TEST_LOG"; }
    security() { printf '%s\\n' 'new-secret-id'; }
    revoke_superseded_client_secrets() { printf 'revoke %s\\n' "$3" >>"$TEST_LOG"; }
    ensure_connector_credentials token identity
  `, "test", bootstrap], { env: { ...process.env, TEST_LOG: log } });

  const events = readFileSync(log, "utf8").trim().split("\n");
  const createPayload = JSON.parse(events[0].slice("create ".length));
  assert.equal(createPayload.ttl, 604800);
  assert.deepEqual(events.slice(1), [
    "store client-id",
    "store client-secret",
    "store client-secret-id",
    "revoke new-secret-id"
  ]);
});

test("credential validation retains seven-day credentials and rejects longer TTLs", () => {
  const tokenPayload = Buffer.from(JSON.stringify({ identityId: "identity" })).toString("base64url");
  const command = `
    source "$1"
    keychain_optional() {
      case "$1" in
        client-id) printf '%s\\n' client ;;
        client-secret) printf '%s\\n' prefix-value ;;
        client-secret-id) printf '%s\\n' current ;;
      esac
    }
    admin_api() {
      jq -cn --argjson ttl "$TEST_TTL" '{clientSecretData:[{id:"current",isClientSecretRevoked:false,clientSecretTTL:$ttl,clientSecretPrefix:"prefix"}]}'
    }
    curl() { cat >/dev/null; printf '{"accessToken":"header.%s.signature"}\\n' "$TEST_JWT_PAYLOAD"; }
    if credentials_are_valid token identity; then printf valid; else printf invalid; fi
  `;
  const runWithTtl = (ttl) => execFileSync("/bin/bash", ["-c", command, "test", bootstrap], {
    encoding: "utf8",
    env: { ...process.env, TEST_TTL: String(ttl), TEST_JWT_PAYLOAD: tokenPayload }
  });

  assert.equal(runWithTtl(604800), "valid");
  assert.equal(runWithTtl(31536000), "invalid");
});

test("Universal Auth uses attach for a fresh identity and update for an existing identity", () => {
  const base = mkdtempSync(join(tmpdir(), "local-dev-env-bootstrap-"));
  const command = `
    source "$1"
    universal_auth_is_attached() { [[ "$TEST_ATTACHED" == "true" ]]; }
    admin_api() { printf '%s\\t%s\\n' "$2" "$4" >>"$TEST_LOG"; printf '%s\\n' '{"identityUniversalAuth":{"clientId":"test"}}'; }
    configure_universal_auth token identity >/dev/null
  `;
  const freshLog = join(base, "fresh.log");
  const existingLog = join(base, "existing.log");
  execFileSync("/bin/bash", ["-c", command, "test", bootstrap], {
    env: { ...process.env, TEST_ATTACHED: "false", TEST_LOG: freshLog }
  });
  execFileSync("/bin/bash", ["-c", command, "test", bootstrap], {
    env: { ...process.env, TEST_ATTACHED: "true", TEST_LOG: existingLog }
  });
  const [freshMethod, freshPayload] = readFileSync(freshLog, "utf8").trim().split("\t");
  const [existingMethod, existingPayload] = readFileSync(existingLog, "utf8").trim().split("\t");
  assert.equal(freshMethod, "POST");
  assert.equal(existingMethod, "PATCH");
  for (const payload of [JSON.parse(freshPayload), JSON.parse(existingPayload)]) {
    assert.equal(payload.accessTokenTTL, 900);
    assert.equal(payload.accessTokenMaxTTL, 900);
  }
});

test("membership reconciliation creates missing access and replaces non-member roles", () => {
  const base = mkdtempSync(join(tmpdir(), "local-dev-env-bootstrap-"));
  const command = `
    source "$1"
    admin_api() {
      if [[ "$2" == "GET" ]]; then printf '%s\\n' "$TEST_MEMBERSHIPS"
      else printf '%s %s\\n' "$2" "$3" >>"$TEST_LOG"
      fi
    }
    ensure_membership token project identity
  `;
  const missingLog = join(base, "missing.log");
  const wrongRoleLog = join(base, "wrong-role.log");
  const memberLog = join(base, "member.log");
  execFileSync("/bin/bash", ["-c", command, "test", bootstrap], {
    env: { ...process.env, TEST_LOG: missingLog, TEST_MEMBERSHIPS: '{"identityMemberships":[]}' }
  });
  execFileSync("/bin/bash", ["-c", command, "test", bootstrap], {
    env: { ...process.env, TEST_LOG: wrongRoleLog, TEST_MEMBERSHIPS: '{"identityMemberships":[{"identityId":"identity","roles":[{"role":"admin","isTemporary":false}]}]}' }
  });
  execFileSync("/bin/bash", ["-c", command, "test", bootstrap], {
    env: { ...process.env, TEST_LOG: memberLog, TEST_MEMBERSHIPS: '{"identityMemberships":[{"identityId":"identity","roles":[{"role":"member","isTemporary":false}]}]}' }
  });
  assert.equal(readFileSync(missingLog, "utf8").trim(), "POST /projects/project/identity-memberships/identity");
  assert.equal(readFileSync(wrongRoleLog, "utf8").trim(), "PATCH /projects/project/identity-memberships/identity");
  assert.throws(() => readFileSync(memberLog), { code: "ENOENT" });
});

test("sensitive curl and Keychain values travel through input streams, not child arguments", () => {
  const base = mkdtempSync(join(tmpdir(), "local-dev-env-bootstrap-"));
  const curlArgs = join(base, "curl-args.log");
  const curlInput = join(base, "curl-input.log");
  const securityArgs = join(base, "security-args.log");
  const securityInput = join(base, "security-input.log");
  execFileSync("/bin/bash", ["-c", `
    source "$1"
    curl() { printf '%s\\n' "$*" >"$CURL_ARGS"; cat >"$CURL_INPUT"; }
    security() { printf '%s\\n' "$*" >"$SECURITY_ARGS"; cat >"$SECURITY_INPUT"; }
    curl_json POST https://example.invalid token-sentinel payload-sentinel >/dev/null
    keychain_put client-secret keychain-sentinel
  `, "test", bootstrap], { env: { ...process.env, CURL_ARGS: curlArgs, CURL_INPUT: curlInput, SECURITY_ARGS: securityArgs, SECURITY_INPUT: securityInput } });

  assert.doesNotMatch(readFileSync(curlArgs, "utf8"), /token-sentinel|payload-sentinel/);
  assert.equal(readFileSync(curlInput, "utf8").trim(), "payload-sentinel");
  assert.doesNotMatch(readFileSync(securityArgs, "utf8"), /keychain-sentinel/);
  assert.equal(readFileSync(securityInput, "utf8"), "keychain-sentinel\nkeychain-sentinel\n");
});
