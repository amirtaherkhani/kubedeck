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

{{- define "platform-storage.pgadminName" -}}
{{- .Values.pgadmin.name | default "pgadmin" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.pgadminLabels" -}}
app.kubernetes.io/name: pgadmin
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: database-admin
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

{{- define "platform-storage.kafkaName" -}}
{{- .Values.kafka.name | default "kafka" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.kafkaLabels" -}}
app.kubernetes.io/name: kafka
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: event-stream
{{- end -}}

{{- define "platform-storage.kafkaUiName" -}}
{{- .Values.kafkaUi.name | default "kafka-ui" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.kafkaUiLabels" -}}
app.kubernetes.io/name: kafka-ui
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: event-stream-admin
{{- end -}}

{{- define "platform-storage.mongodbName" -}}
{{- .Values.mongodb.name | default "mongodb" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.mongodbLabels" -}}
app.kubernetes.io/name: mongodb
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: document-database
{{- end -}}

{{- define "platform-storage.mongokuName" -}}
{{- .Values.mongoku.name | default "mongoku" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.mongokuLabels" -}}
app.kubernetes.io/name: mongoku
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: document-database-admin
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

{{- define "platform-storage.natsUiName" -}}
{{- .Values.natsUi.name | default "nats-ui" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.natsUiLabels" -}}
app.kubernetes.io/name: nats-ui
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: message-bus-admin
{{- end -}}

{{- define "platform-storage.minioName" -}}
{{- .Values.minio.name | default "minio" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.minioLabels" -}}
app.kubernetes.io/name: minio
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: object-storage
{{- end -}}

{{- define "platform-storage.minioInitName" -}}
{{- printf "%s-init" (include "platform-storage.minioName" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.jaegerName" -}}
{{- .Values.jaeger.name | default "jaeger" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.jaegerLabels" -}}
app.kubernetes.io/name: jaeger
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: tracing
{{- end -}}

{{- define "platform-storage.jaegerUiName" -}}
{{- printf "%s-ui" (include "platform-storage.jaegerName" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.jaegerUiLabels" -}}
app.kubernetes.io/name: jaeger-ui
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: tracing-admin
{{- end -}}

{{- define "platform-storage.mailpitName" -}}
{{- .Values.mailpit.name | default "mailpit" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.mailpitLabels" -}}
app.kubernetes.io/name: mailpit
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: mail-testing
{{- end -}}

{{- define "platform-storage.mailpitUiName" -}}
{{- printf "%s-ui" (include "platform-storage.mailpitName" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.mailpitUiLabels" -}}
app.kubernetes.io/name: mailpit-ui
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: mail-testing-admin
{{- end -}}

{{- define "platform-storage.schemaRegistryName" -}}
{{- .Values.schemaRegistry.name | default "schema-registry" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.schemaRegistryLabels" -}}
app.kubernetes.io/name: schema-registry
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: schema-registry
{{- end -}}

{{- define "platform-storage.schemaRegistryUiName" -}}
{{- printf "%s-ui" (include "platform-storage.schemaRegistryName" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.schemaRegistryUiLabels" -}}
app.kubernetes.io/name: schema-registry-ui
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: schema-registry-admin
{{- end -}}

{{- define "platform-storage.grpcuiName" -}}
{{- .Values.grpcui.name | default "grpcui" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "platform-storage.grpcuiLabels" -}}
app.kubernetes.io/name: grpcui
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: grpc-tooling
{{- end -}}
