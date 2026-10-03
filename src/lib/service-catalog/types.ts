/** Declarative contract version; live agent snapshots have a separate schema. */
export const SERVICE_CATALOG_API_VERSION = 'catalog.kubedeck.io/v1alpha1' as const;

export type ModuleOwnership = 'managed' | 'external';
export type ComponentType =
  | 'database'
  | 'broker'
  | 'microservice'
  | 'web-app'
  | 'worker'
  | 'infrastructure'
  | 'open-source-tool'
  | 'other';

export interface ModuleMetadata {
  id: string;
  displayName: string;
  category: string;
  ownership: ModuleOwnership;
  purpose: string;
  documentation?: string;
  tags?: string[];
}

export interface ExternalConnection {
  name: string;
  targetRef: string;
  protocol?: string;
}

export interface WorkloadRef {
  apiVersion: string;
  kind: string;
  name: string;
  namespace: string;
  labelSelector?: Record<string, string>;
}

export interface Endpoint {
  name: string;
  protocol: 'http' | 'https' | 'tcp' | 'grpc' | 'amqp' | 'other';
  addressRef: string;
  portName?: string;
}

export interface HealthCheck {
  name: string;
  type: 'kubernetes-readiness' | 'kubernetes-liveness' | 'http' | 'tcp';
  targetRef: string;
  path?: string;
  expectedStatus?: number;
}

export interface ObservabilityRefs {
  dashboards?: string[];
  metrics?: string[];
  logs?: string[];
}

export interface InfisicalSecretRef {
  provider: 'infisical';
  projectRef: string;
  environmentRef: string;
  path: string;
  secretName?: string;
  keys?: string[];
}

export interface ServiceComponent {
  id: string;
  type: ComponentType;
  purpose: string;
  provides?: string[];
  requires?: string[];
  externalConnections?: ExternalConnection[];
  workloadRefs?: WorkloadRef[];
  endpoints?: Endpoint[];
  healthChecks?: HealthCheck[];
  observability?: ObservabilityRefs;
  secretRefs?: InfisicalSecretRef[];
}

export interface ServiceModule {
  apiVersion: typeof SERVICE_CATALOG_API_VERSION;
  kind: 'ServiceModule';
  module: ModuleMetadata;
  components: ServiceComponent[];
}

export interface ChartRef {
  source: string;
  name: string;
  version: string;
  localPath?: string;
}

export interface Installation {
  moduleId: string;
  releaseName: string;
  namespace: string;
  chart: ChartRef;
  valuesFiles?: string[];
}

export interface InstallationProfile {
  apiVersion: typeof SERVICE_CATALOG_API_VERSION;
  kind: 'InstallationProfile';
  profile: {
    id: string;
    displayName: string;
    target: 'local-single-node';
  };
  installations: Installation[];
}
