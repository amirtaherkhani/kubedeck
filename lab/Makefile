RELEASE ?= all

.PHONY: core-validate core-apply docker-desktop-traefik-crds platform-storage-test macos-dns macos-dns-check macos-trust-tls infisical-bootstrap infisical-admin-bootstrap kubedeck-env-bootstrap kubedeck-env-test kubedeck-env-doctor local-dev-env-bootstrap local-dev-env-test local-dev-env-doctor helm-inventory helm-repos helm-status helm-validate helm-apply helm-values helm-remove helm-lint nats-ha-test tempo-smoke

core-validate:
	kubectl apply --dry-run=server -f core/namespaces/namespaces.yaml
	kubectl apply --dry-run=server -f core/nodes/inotify-capacity.yaml
	kubectl apply --dry-run=server -f core/dns/coredns-custom.yaml
	kubectl apply --dry-run=server -f core/dns/service.yaml
	kubectl apply --dry-run=server -f core/ingress/tlsstore.yaml
	kubectl apply --dry-run=server -f core/ingress/traefik-helmchartconfig.yaml
	kubectl apply --dry-run=server -f core/tls/clusterissuer.yaml
	kubectl apply --dry-run=server -f core/tls/certificates.yaml

core-apply:
	./scripts/core-bootstrap.sh

docker-desktop-traefik-crds:
	./scripts/docker-desktop-traefik-crds.sh

platform-storage-test:
	./apps/platform/storage/tests/test_infisicalsecret_templates.sh

macos-dns:
	./core/host/macos/configure-local-dev-dns-forwarding.sh
	./core/host/macos/configure-local-dev-resolver.sh

macos-dns-check:
	./core/host/macos/configure-local-dev-resolver.sh --dry-run

macos-trust-tls:
	./core/host/macos/trust-local-dev-tls.sh

infisical-bootstrap:
	./scripts/infisical-bootstrap.sh

infisical-admin-bootstrap:
	./scripts/infisical-admin-bootstrap.sh

kubedeck-env-bootstrap:
	./scripts/local-dev-env-bootstrap.sh

local-dev-env-bootstrap: kubedeck-env-bootstrap

kubedeck-env-test:
	npm test --prefix tools/kubedeck-env-mcp

local-dev-env-test: kubedeck-env-test

kubedeck-env-doctor:
	NODE_USE_SYSTEM_CA=1 kubedeck-env-mcp doctor

local-dev-env-doctor: kubedeck-env-doctor

helm-inventory:
	./scripts/platform-helm.sh inventory

helm-repos:
	./scripts/platform-helm.sh repos

helm-status:
	./scripts/platform-helm.sh status

helm-validate:
	./scripts/platform-helm.sh validate "$(RELEASE)"

helm-apply:
	./scripts/platform-helm.sh apply "$(RELEASE)"

helm-values:
	@test "$(RELEASE)" != "all" || (echo "Set RELEASE to one managed release." && exit 1)
	./scripts/platform-helm.sh values "$(RELEASE)"

helm-lint:
	helm lint apps/dev/kubedeck apps/dev/kubedeck-agent apps/platform/storage apps/observability/grafana

helm-remove:
	@test "$(RELEASE)" != "all" || (echo "Set RELEASE to one managed release." && exit 1)
	CONFIRM_UNINSTALL="$(CONFIRM_UNINSTALL)" ./scripts/platform-helm.sh remove "$(RELEASE)"

nats-ha-test:
	./scripts/test-nats-ha.sh

tempo-smoke:
	./scripts/tempo-smoke.sh
