{{/*
Common labels stamped onto every resource. Helm 3 convention —
keeps `kubectl get all -l app.kubernetes.io/instance=<release>`
working as a "what did this release create" query.
*/}}
{{- define "havoc.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{/*
Per-component label set. Used both on the resource itself and as
the selector for Deployments/DaemonSets/Services.
*/}}
{{- define "havoc.componentLabels" -}}
{{ include "havoc.labels" . }}
app.kubernetes.io/component: {{ .component }}
app: {{ .name }}
{{- end -}}

{{/*
Postgres + Redis env block. Switches between secret-backed (AWS,
when secrets.enabled) and inline values (kind defaults). Used by
control and recorder. Pass the root scope as `.`.
*/}}
{{- define "havoc.dataPlaneEnv" -}}
{{- if .Values.secrets.enabled }}
- name: HAVOC_POSTGRES_DSN
  valueFrom:
    secretKeyRef:
      name: {{ .Values.secrets.name }}
      key: postgresDSN
- name: HAVOC_REDIS_ADDR
  valueFrom:
    secretKeyRef:
      name: {{ .Values.secrets.name }}
      key: redisAddr
- name: HAVOC_REDIS_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ .Values.secrets.name }}
      key: redisPassword
      optional: true
{{- else }}
- name: HAVOC_POSTGRES_DSN
  value: {{ .Values.global.postgresDSN | quote }}
- name: HAVOC_REDIS_ADDR
  value: {{ .Values.global.redisAddr | quote }}
{{- if .Values.global.redisPassword }}
- name: HAVOC_REDIS_PASSWORD
  value: {{ .Values.global.redisPassword | quote }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
Redis-only env block (no Postgres). Used by the agent.
*/}}
{{- define "havoc.redisEnv" -}}
{{- if .Values.secrets.enabled }}
- name: HAVOC_REDIS_ADDR
  valueFrom:
    secretKeyRef:
      name: {{ .Values.secrets.name }}
      key: redisAddr
- name: HAVOC_REDIS_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ .Values.secrets.name }}
      key: redisPassword
      optional: true
{{- else }}
- name: HAVOC_REDIS_ADDR
  value: {{ .Values.global.redisAddr | quote }}
{{- if .Values.global.redisPassword }}
- name: HAVOC_REDIS_PASSWORD
  value: {{ .Values.global.redisPassword | quote }}
{{- end }}
{{- end }}
{{- end -}}
