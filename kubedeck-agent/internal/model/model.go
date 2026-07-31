package model

import "time"

const SchemaVersion = "kubedeck.io/v1alpha1"

type Snapshot struct {
	SchemaVersion string            `json:"schemaVersion"`
	GeneratedAt   time.Time         `json:"generatedAt"`
	Cluster       Cluster           `json:"cluster"`
	Summary       Summary           `json:"summary"`
	DNS           DNSProfile        `json:"dns"`
	Nodes         []Node            `json:"nodes"`
	Pods          []Pod             `json:"pods"`
	Workloads     []Workload        `json:"workloads"`
	Services      []Service         `json:"services"`
	Ingresses     []Ingress         `json:"ingresses"`
	Volumes       []Volume          `json:"volumes"`
	Events        []KubernetesEvent `json:"events"`
}

type Cluster struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	KubernetesVersion string `json:"kubernetesVersion"`
	Platform          string `json:"platform,omitempty"`
}

type Summary struct {
	Nodes             int `json:"nodes"`
	ReadyNodes        int `json:"readyNodes"`
	Namespaces        int `json:"namespaces"`
	Pods              int `json:"pods"`
	ReadyPods         int `json:"readyPods"`
	Workloads         int `json:"workloads"`
	ReadyWorkloads    int `json:"readyWorkloads"`
	Services          int `json:"services"`
	ReadyServices     int `json:"readyServices"`
	Ingresses         int `json:"ingresses"`
	PersistentVolumes int `json:"persistentVolumes"`
	WarningEvents     int `json:"warningEvents"`
}

type DNSProfile struct {
	Provider       string   `json:"provider"`
	ServiceName    string   `json:"serviceName,omitempty"`
	ServiceDNS     string   `json:"serviceDNS,omitempty"`
	ServiceIP      string   `json:"serviceIP,omitempty"`
	ClusterDomain  string   `json:"clusterDomain"`
	Ports          []Port   `json:"ports"`
	SearchPath     []string `json:"searchPath"`
	Ready          bool     `json:"ready"`
	ReadyEndpoints int      `json:"readyEndpoints"`
	TotalEndpoints int      `json:"totalEndpoints"`
}

type Node struct {
	Name             string            `json:"name"`
	Role             string            `json:"role"`
	Ready            bool              `json:"ready"`
	Status           string            `json:"status"`
	InternalIP       string            `json:"internalIP,omitempty"`
	Hostname         string            `json:"hostname,omitempty"`
	KubeletVersion   string            `json:"kubeletVersion"`
	OSImage          string            `json:"osImage"`
	OperatingSystem  string            `json:"operatingSystem"`
	Architecture     string            `json:"architecture"`
	ContainerRuntime string            `json:"containerRuntime"`
	CreatedAt        time.Time         `json:"createdAt"`
	UptimeSeconds    int64             `json:"uptimeSeconds"`
	LastHeartbeatAt  *time.Time        `json:"lastHeartbeatAt,omitempty"`
	Capacity         NodeCapacity      `json:"capacity"`
	Usage            NodeUsage         `json:"usage"`
	Conditions       []NodeCondition   `json:"conditions"`
	Unschedulable    bool              `json:"unschedulable"`
	PodCIDR          string            `json:"podCIDR,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
}

type NodeCapacity struct {
	CPUMilli              int64 `json:"cpuMilli"`
	MemoryBytes           int64 `json:"memoryBytes"`
	EphemeralStorageBytes int64 `json:"ephemeralStorageBytes"`
	Pods                  int64 `json:"pods"`
}

type NodeUsage struct {
	CPUMilli                       int64   `json:"cpuMilli"`
	CPUPercent                     float64 `json:"cpuPercent"`
	MemoryBytes                    int64   `json:"memoryBytes"`
	MemoryPercent                  float64 `json:"memoryPercent"`
	Pods                           int     `json:"pods"`
	PodAllocationPercent           float64 `json:"podAllocationPercent"`
	EphemeralStorageRequestedBytes int64   `json:"ephemeralStorageRequestedBytes"`
	EphemeralStorageRequestPercent float64 `json:"ephemeralStorageRequestPercent"`
	MetricsAvailable               bool    `json:"metricsAvailable"`
}

type NodeCondition struct {
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	Reason             string     `json:"reason,omitempty"`
	Message            string     `json:"message,omitempty"`
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
}

type Pod struct {
	UID        string            `json:"uid"`
	Namespace  string            `json:"namespace"`
	Name       string            `json:"name"`
	NodeName   string            `json:"nodeName,omitempty"`
	Phase      string            `json:"phase"`
	Status     string            `json:"status"`
	Reason     string            `json:"reason,omitempty"`
	Ready      bool              `json:"ready"`
	Restarts   int32             `json:"restarts"`
	CreatedAt  time.Time         `json:"createdAt"`
	StartedAt  *time.Time        `json:"startedAt,omitempty"`
	OwnerKind  string            `json:"ownerKind,omitempty"`
	OwnerName  string            `json:"ownerName,omitempty"`
	PodIP      string            `json:"podIP,omitempty"`
	HostIP     string            `json:"hostIP,omitempty"`
	Containers []Container       `json:"containers"`
	Usage      PodUsage          `json:"usage"`
	Labels     map[string]string `json:"labels,omitempty"`
}

type Container struct {
	Name         string     `json:"name"`
	Image        string     `json:"image"`
	Ready        bool       `json:"ready"`
	RestartCount int32      `json:"restartCount"`
	State        string     `json:"state"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
}

type PodUsage struct {
	CPUMilli         int64 `json:"cpuMilli"`
	MemoryBytes      int64 `json:"memoryBytes"`
	MetricsAvailable bool  `json:"metricsAvailable"`
}

type Workload struct {
	UID            string            `json:"uid"`
	Kind           string            `json:"kind"`
	Namespace      string            `json:"namespace"`
	Name           string            `json:"name"`
	Desired        int32             `json:"desired"`
	Ready          int32             `json:"ready"`
	Available      int32             `json:"available"`
	Status         string            `json:"status"`
	CreatedAt      time.Time         `json:"createdAt"`
	LastDeployedAt *time.Time        `json:"lastDeployedAt,omitempty"`
	Selector       map[string]string `json:"selector,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
}

type ObjectReference struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type Service struct {
	UID            string            `json:"uid"`
	Namespace      string            `json:"namespace"`
	Name           string            `json:"name"`
	Category       string            `json:"category"`
	Type           string            `json:"type"`
	Status         string            `json:"status"`
	ClusterDNS     string            `json:"clusterDNS"`
	ClusterIP      string            `json:"clusterIP,omitempty"`
	ExternalIPs    []string          `json:"externalIPs,omitempty"`
	ExternalURLs   []string          `json:"externalURLs,omitempty"`
	Ports          []Port            `json:"ports"`
	Selector       map[string]string `json:"selector,omitempty"`
	ReadyEndpoints int               `json:"readyEndpoints"`
	TotalEndpoints int               `json:"totalEndpoints"`
	ReadyPods      int               `json:"readyPods"`
	TotalPods      int               `json:"totalPods"`
	Workloads      []ObjectReference `json:"workloads"`
	LastDeployedAt *time.Time        `json:"lastDeployedAt,omitempty"`
	UptimeSeconds  int64             `json:"uptimeSeconds"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
}

type Port struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol"`
	Port       int32  `json:"port"`
	TargetPort string `json:"targetPort,omitempty"`
	NodePort   int32  `json:"nodePort,omitempty"`
}

type Ingress struct {
	UID       string         `json:"uid"`
	Namespace string         `json:"namespace"`
	Name      string         `json:"name"`
	ClassName string         `json:"className,omitempty"`
	Routes    []IngressRoute `json:"routes"`
	TLSHosts  []string       `json:"tlsHosts,omitempty"`
	Addresses []string       `json:"addresses,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

type IngressRoute struct {
	Host        string `json:"host,omitempty"`
	Path        string `json:"path"`
	PathType    string `json:"pathType,omitempty"`
	ServiceName string `json:"serviceName"`
	ServicePort string `json:"servicePort"`
	URL         string `json:"url,omitempty"`
}

type Volume struct {
	UID           string            `json:"uid"`
	Namespace     string            `json:"namespace,omitempty"`
	Name          string            `json:"name"`
	Kind          string            `json:"kind"`
	Status        string            `json:"status"`
	StorageClass  string            `json:"storageClass,omitempty"`
	CapacityBytes int64             `json:"capacityBytes"`
	AccessModes   []string          `json:"accessModes,omitempty"`
	VolumeName    string            `json:"volumeName,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	Labels        map[string]string `json:"labels,omitempty"`
}

type KubernetesEvent struct {
	UID                string     `json:"uid"`
	Namespace          string     `json:"namespace,omitempty"`
	Name               string     `json:"name"`
	Type               string     `json:"type"`
	Reason             string     `json:"reason,omitempty"`
	Message            string     `json:"message,omitempty"`
	RegardingKind      string     `json:"regardingKind,omitempty"`
	RegardingNamespace string     `json:"regardingNamespace,omitempty"`
	RegardingName      string     `json:"regardingName,omitempty"`
	Count              int32      `json:"count"`
	FirstSeenAt        *time.Time `json:"firstSeenAt,omitempty"`
	LastSeenAt         *time.Time `json:"lastSeenAt,omitempty"`
}
