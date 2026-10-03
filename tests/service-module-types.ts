import type {
  InstallationProfile,
  ServiceModule,
} from '../src/lib/service-catalog/types';

const managed: ServiceModule = {
  apiVersion: 'catalog.kubedeck.io/v1alpha1',
  kind: 'ServiceModule',
  module: {
    id: 'example-module',
    displayName: 'Example Module',
    category: 'example',
    ownership: 'managed',
    purpose: 'Compile-time contract fixture',
  },
  components: [{ id: 'example-component', type: 'other', purpose: 'Example' }],
};

const profile: InstallationProfile = {
  apiVersion: 'catalog.kubedeck.io/v1alpha1',
  kind: 'InstallationProfile',
  profile: { id: 'local-single-node', displayName: 'Local Single-Node', target: 'local-single-node' },
  installations: [{
    moduleId: managed.module.id,
    releaseName: 'example-module',
    namespace: 'example',
    chart: { source: 'example-repository', name: 'example-chart', version: '0.0.1' },
  }],
};

export { managed, profile };
