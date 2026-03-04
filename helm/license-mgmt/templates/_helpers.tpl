{{- /*
Template helpers - included in all templates
*/ -}}
{{/*
Expand the name of the chart.
*/}}
{{- define "license-mgmt.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "license-mgmt.fullname" -}}
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
{{- define "license-mgmt.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "license-mgmt.labels" -}}
helm.sh/chart: {{ include "license-mgmt.chart" . }}
{{ include "license-mgmt.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "license-mgmt.selectorLabels" -}}
app.kubernetes.io/name: {{ include "license-mgmt.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app: {{ include "license-mgmt.name" . }}-api
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "license-mgmt.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "license-mgmt.fullname" . ) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
