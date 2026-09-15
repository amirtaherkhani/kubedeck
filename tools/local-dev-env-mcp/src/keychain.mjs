import { execFileSync } from "node:child_process";

function keychainValue(service, account, profile) {
  try {
    return execFileSync("/usr/bin/security", ["find-generic-password", "-s", service, "-a", account, "-w"], {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"]
    }).trim();
  } catch (error) {
    if (error?.status === 44) return "";
    throw new Error(`macOS Keychain lookup failed for Infisical profile ${profile}`);
  }
}

export function loadCredentials(profile) {
  const service = `my-home-lab.infisical.agent.${profile}`;
  const clientId = keychainValue(service, "client-id", profile);
  const clientSecret = keychainValue(service, "client-secret", profile);
  if (!clientId || !clientSecret) {
    throw new Error(`Infisical agent profile ${profile} is not configured; run the owning project's access bootstrap`);
  }
  return { clientId, clientSecret };
}
