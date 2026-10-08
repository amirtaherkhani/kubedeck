{{- define "platform-storage.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.labels" -}}
app.kubernetes.io/name: {{ include "platform-storage.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "platform-storage.postgresqlName" -}}
{{- .Values.postgresql.name | default "postgresql" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.postgresqlLabels" -}}
app.kubernetes.io/name: postgresql
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: database
{{- end -}}

{{- define "platform-storage.redisName" -}}
{{- .Values.redis.name | default "redis" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.redisLabels" -}}
app.kubernetes.io/name: redis
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: cache
{{- end -}}

{{- define "platform-storage.rabbitmqName" -}}
{{- .Values.rabbitmq.name | default "rabbitmq" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.rabbitmqLabels" -}}
app.kubernetes.io/name: rabbitmq
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: message-broker
{{- end -}}

{{- define "platform-storage.natsName" -}}
{{- .Values.nats.name | default "nats" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.natsHeadlessName" -}}
{{- printf "%s-headless" (include "platform-storage.natsName" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.natsLabels" -}}
app.kubernetes.io/name: nats
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: message-bus
{{- end -}}

{{- define "platform-storage.natsMonitoringName" -}}
{{- printf "%s-monitoring" (include "platform-storage.natsName" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.natsMonitoringLabels" -}}
app.kubernetes.io/name: nats-monitoring
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: message-bus-monitoring
{{- end -}}

{{- define "platform-storage.natsMetricsName" -}}
{{- printf "%s-metrics" (include "platform-storage.natsName" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.natsMetricsLabels" -}}
app.kubernetes.io/name: nats-metrics
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: message-bus-metrics
{{- end -}}
