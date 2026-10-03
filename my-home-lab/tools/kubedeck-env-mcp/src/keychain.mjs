import { execFileSync } from "node:child_process";

const KEYCHAIN_SERVICE = process.env.LOCAL_DEV_ENV_KEYCHAIN_SERVICE ?? "my-home-lab.infisical.agent.local-dev";

function keychainValue(account) {
  try {
    return execFileSync("/usr/bin/security", ["find-generic-password", "-s", KEYCHAIN_SERVICE, "-a", account, "-w"], {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"]
    }).trim();
  } catch (error) {
    if (error?.status === 44) return "";
    throw new Error("macOS Keychain lookup failed for the local development Infisical identity");
  }
}

export function loadCredentials() {
  const clientId = keychainValue("client-id");
  const clientSecret = keychainValue("client-secret");
  if (!clientId || !clientSecret) {
    throw new Error("local development Infisical credentials are not configured; run make local-dev-env-bootstrap");
  }
  return { clientId, clientSecret };
}
