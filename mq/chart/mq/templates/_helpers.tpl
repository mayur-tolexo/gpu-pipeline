{{- define "mq.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "mq.fullname" -}}
{{- printf "%s-%s" (include "mq.name" .) .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
