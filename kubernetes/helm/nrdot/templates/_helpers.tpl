{{/*
Expand the name of the chart.
*/}}
{{- define "nrdot.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "nrdot.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "nrdot.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "nrdot.labels" -}}
helm.sh/chart: {{ include "nrdot.chart" . }}
{{ include "nrdot.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: nrdot-host
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "nrdot.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nrdot.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Component labels
*/}}
{{- define "nrdot.componentLabels" -}}
{{ include "nrdot.labels" . }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Component selector labels
*/}}
{{- define "nrdot.componentSelectorLabels" -}}
{{ include "nrdot.selectorLabels" . }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "nrdot.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (printf "%s-%s" (include "nrdot.fullname" .) .component) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the namespace name
*/}}
{{- define "nrdot.namespace" -}}
{{- if .Values.namespace.create }}
{{- default .Values.namespace.name .Release.Namespace }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Get the New Relic license key
*/}}
{{- define "nrdot.newrelicLicenseKey" -}}
{{- if .Values.newrelic.licenseKey }}
{{- .Values.newrelic.licenseKey }}
{{- else if .Values.newrelic.licenseKeySecretName }}
valueFrom:
  secretKeyRef:
    name: {{ .Values.newrelic.licenseKeySecretName }}
    key: {{ .Values.newrelic.licenseKeySecretKey }}
{{- else }}
{{- fail "Either newrelic.licenseKey or newrelic.licenseKeySecretName must be set" }}
{{- end }}
{{- end }}

{{/*
Get the API auth token
*/}}
{{- define "nrdot.apiAuthToken" -}}
{{- if .Values.apiServer.config.authToken }}
{{- .Values.apiServer.config.authToken }}
{{- else if .Values.apiServer.config.authTokenSecretName }}
valueFrom:
  secretKeyRef:
    name: {{ .Values.apiServer.config.authTokenSecretName }}
    key: {{ .Values.apiServer.config.authTokenSecretKey }}
{{- end }}
{{- end }}

{{/*
Get the New Relic OTLP endpoint
*/}}
{{- define "nrdot.newrelicOtlpEndpoint" -}}
{{- if .Values.newrelic.euDatacenter }}
{{- "otlp.eu01.nr-data.net:4317" }}
{{- else }}
{{- .Values.newrelic.otlpEndpoint }}
{{- end }}
{{- end }}

{{/*
Image pull secrets
*/}}
{{- define "nrdot.imagePullSecrets" -}}
{{- $secrets := list }}
{{- if .Values.global.imagePullSecrets }}
{{- range .Values.global.imagePullSecrets }}
{{- $secrets = append $secrets . }}
{{- end }}
{{- end }}
{{- range .component.imagePullSecrets }}
{{- $secrets = append $secrets . }}
{{- end }}
{{- if $secrets }}
imagePullSecrets:
{{- range $secrets | uniq }}
- name: {{ . }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Container image
*/}}
{{- define "nrdot.image" -}}
{{- $registry := .component.image.registry | default .Values.global.imageRegistry | default "" -}}
{{- $repository := .component.image.repository -}}
{{- $tag := .component.image.tag | default .Chart.AppVersion -}}
{{- if $registry -}}
{{- printf "%s/%s:%s" $registry $repository $tag -}}
{{- else -}}
{{- printf "%s:%s" $repository $tag -}}
{{- end -}}
{{- end }}

{{/*
Node selector
*/}}
{{- define "nrdot.nodeSelector" -}}
{{- $nodeSelector := .Values.global.nodeSelector }}
{{- if .component.nodeSelector }}
{{- $nodeSelector = .component.nodeSelector }}
{{- end }}
{{- if $nodeSelector }}
nodeSelector:
{{ toYaml $nodeSelector | indent 2 }}
{{- end }}
{{- end }}

{{/*
Tolerations
*/}}
{{- define "nrdot.tolerations" -}}
{{- $tolerations := .Values.global.tolerations }}
{{- if .component.tolerations }}
{{- $tolerations = concat $tolerations .component.tolerations | uniq }}
{{- end }}
{{- if $tolerations }}
tolerations:
{{ toYaml $tolerations | indent 2 }}
{{- end }}
{{- end }}

{{/*
Affinity
*/}}
{{- define "nrdot.affinity" -}}
{{- if .component.affinity }}
affinity:
{{ toYaml .component.affinity | indent 2 }}
{{- else }}
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app.kubernetes.io/component
            operator: In
            values:
            - {{ .componentName }}
        topologyKey: kubernetes.io/hostname
{{- end }}
{{- end }}

{{/*
Pod security context
*/}}
{{- define "nrdot.podSecurityContext" -}}
{{- $context := deepCopy .Values.global.podSecurityContext }}
{{- if .component.podSecurityContext }}
{{- $context = mergeOverwrite $context .component.podSecurityContext }}
{{- end }}
{{- if $context }}
securityContext:
{{ toYaml $context | indent 2 }}
{{- end }}
{{- end }}

{{/*
Container security context
*/}}
{{- define "nrdot.securityContext" -}}
{{- if .component.securityContext }}
securityContext:
{{ toYaml .component.securityContext | indent 2 }}
{{- end }}
{{- end }}

{{/*
Prometheus annotations
*/}}
{{- define "nrdot.prometheusAnnotations" -}}
prometheus.io/scrape: "true"
prometheus.io/port: {{ .port | quote }}
prometheus.io/path: {{ .path | default "/metrics" | quote }}
{{- end }}

{{/*
Checksum annotations
*/}}
{{- define "nrdot.checksumAnnotations" -}}
checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
{{- if .Values.tls.enabled }}
checksum/tls: {{ include (print $.Template.BasePath "/secrets.yaml") . | sha256sum }}
{{- end }}
{{- end }}

{{/*
Environment variables for New Relic
*/}}
{{- define "nrdot.newrelicEnvVars" -}}
- name: NEW_RELIC_LICENSE_KEY
  {{- if .Values.newrelic.licenseKey }}
  value: {{ .Values.newrelic.licenseKey | quote }}
  {{- else }}
  valueFrom:
    secretKeyRef:
      name: {{ .Values.newrelic.licenseKeySecretName }}
      key: {{ .Values.newrelic.licenseKeySecretKey }}
  {{- end }}
- name: NEW_RELIC_OTLP_ENDPOINT
  value: {{ include "nrdot.newrelicOtlpEndpoint" . | quote }}
{{- end }}

{{/*
Common environment variables
*/}}
{{- define "nrdot.commonEnvVars" -}}
- name: NRDOT_LOG_LEVEL
  value: {{ .Values.global.logLevel | default "info" | quote }}
- name: GOMAXPROCS
  valueFrom:
    resourceFieldRef:
      resource: limits.cpu
- name: GOMEMLIMIT
  valueFrom:
    resourceFieldRef:
      resource: limits.memory
{{- end }}

{{/*
Extra environment variables
*/}}
{{- define "nrdot.extraEnvVars" -}}
{{- if .Values.extraEnvVars }}
{{ toYaml .Values.extraEnvVars }}
{{- end }}
{{- if .Values.extraEnvVarsSecret }}
- secretRef:
    name: {{ .Values.extraEnvVarsSecret }}
{{- end }}
{{- end }}

{{/*
Volume mounts for config
*/}}
{{- define "nrdot.configVolumeMounts" -}}
- name: config
  mountPath: /etc/nrdot
  readOnly: true
{{- if .Values.tls.enabled }}
- name: tls-certs
  mountPath: /etc/nrdot/tls
  readOnly: true
{{- end }}
- name: temp
  mountPath: /tmp
- name: cache
  mountPath: /var/cache/nrdot
{{- end }}

{{/*
Volumes for config
*/}}
{{- define "nrdot.configVolumes" -}}
- name: config
  configMap:
    name: {{ include "nrdot.fullname" . }}-config
{{- if .Values.tls.enabled }}
- name: tls-certs
  secret:
    secretName: {{ include "nrdot.fullname" . }}-tls
    defaultMode: 0400
{{- end }}
- name: temp
  emptyDir: {}
- name: cache
  emptyDir: {}
{{- end }}