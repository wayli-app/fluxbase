{{/*
Expand the name of the chart.
*/}}
{{- define "fluxbase.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "fluxbase.fullname" -}}
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
{{- define "fluxbase.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels - Kubernetes standard labels
*/}}
{{- define "fluxbase.labels" -}}
helm.sh/chart: {{ include "fluxbase.chart" . }}
{{ include "fluxbase.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Values.commonLabels }}
{{ toYaml .Values.commonLabels }}
{{- end }}
{{- end }}

{{/*
Selector labels - Used for immutable fields like deployment spec selectors
*/}}
{{- define "fluxbase.selectorLabels" -}}
app.kubernetes.io/name: {{ include "fluxbase.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "fluxbase.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "fluxbase.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return the proper Fluxbase image name
*/}}
{{- define "fluxbase.image" -}}
{{- $registryName := .Values.image.registry -}}
{{- $repositoryName := .Values.image.repository -}}
{{- $separator := ":" -}}
{{- $termination := .Values.image.tag | toString -}}
{{- if .Values.global }}
    {{- if .Values.global.imageRegistry }}
     {{- $registryName = .Values.global.imageRegistry -}}
    {{- end -}}
{{- end -}}
{{- if .Values.image.digest }}
    {{- $separator = "@" -}}
    {{- $termination = .Values.image.digest | toString -}}
{{- end -}}
{{- if $registryName }}
    {{- printf "%s/%s%s%s" $registryName $repositoryName $separator $termination -}}
{{- else -}}
    {{- printf "%s%s%s"  $repositoryName $separator $termination -}}
{{- end -}}
{{- end -}}

{{/*
Return the proper Docker Image Registry Secret Names
*/}}
{{- define "fluxbase.imagePullSecrets" -}}
{{- $pullSecrets := list }}

{{- if .Values.global }}
  {{- range .Values.global.imagePullSecrets }}
    {{- $pullSecrets = append $pullSecrets . }}
  {{- end }}
{{- end }}

{{- range .Values.image.pullSecrets }}
  {{- $pullSecrets = append $pullSecrets . }}
{{- end }}

{{- if (not (empty $pullSecrets)) }}
imagePullSecrets:
{{- range $pullSecrets }}
  - name: {{ . }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
Return the proper secret name for Fluxbase secrets
*/}}
{{- define "fluxbase.secretName" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecret -}}
{{- else }}
    {{- include "fluxbase.fullname" . -}}
{{- end }}
{{- end }}

{{/*
Return the proper PostgreSQL host
*/}}
{{- define "fluxbase.databaseHost" -}}
{{- if .Values.fluxbase.database.host }}
    {{- .Values.fluxbase.database.host -}}
{{- else if eq .Values.postgresql.mode "standalone" }}
    {{- printf "%s-postgresql" (include "fluxbase.fullname" .) -}}
{{- else if eq .Values.postgresql.mode "cnpg" }}
    {{- if .Values.postgresql.cnpg.pooler.enabled }}
        {{- printf "%s-postgresql-pooler-rw" (include "fluxbase.fullname" .) -}}
    {{- else }}
        {{- printf "%s-postgresql-rw" (include "fluxbase.fullname" .) -}}
    {{- end }}
{{- else if eq .Values.postgresql.mode "none" }}
    {{- .Values.externalDatabase.host -}}
{{- else }}
    {{- .Values.externalDatabase.host -}}
{{- end }}
{{- end }}

{{/*
Return the proper PostgreSQL port
*/}}
{{- define "fluxbase.databasePort" -}}
{{- if ne .Values.postgresql.mode "none" }}
    {{- print "5432" -}}
{{- else }}
    {{- .Values.externalDatabase.port -}}
{{- end }}
{{- end }}

{{/*
Return the proper PostgreSQL database name
*/}}
{{- define "fluxbase.databaseName" -}}
{{- if ne .Values.postgresql.mode "none" }}
    {{- .Values.postgresql.auth.database -}}
{{- else }}
    {{- .Values.externalDatabase.database -}}
{{- end }}
{{- end }}

{{/*
Return the proper PostgreSQL runtime username
*/}}
{{- define "fluxbase.databaseUser" -}}
{{- if .Values.fluxbase.database.user }}
    {{- .Values.fluxbase.database.user -}}
{{- else if ne .Values.postgresql.mode "none" }}
    {{- .Values.postgresql.auth.username -}}
{{- else }}
    {{- .Values.externalDatabase.user -}}
{{- end }}
{{- end }}

{{/*
Return the proper PostgreSQL password secret name
*/}}
{{- define "fluxbase.databaseSecretName" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecret -}}
{{- else if eq .Values.postgresql.mode "standalone" }}
    {{- printf "%s-postgresql" (include "fluxbase.fullname" .) -}}
{{- else if eq .Values.postgresql.mode "cnpg" }}
    {{- printf "%s-postgresql-app" (include "fluxbase.fullname" .) -}}
{{- else if eq .Values.postgresql.mode "none" }}
    {{- if .Values.externalDatabase.existingSecret }}
        {{- .Values.externalDatabase.existingSecret -}}
    {{- else }}
        {{- printf "%s" (include "fluxbase.fullname" .) -}}
    {{- end }}
{{- else }}
    {{- printf "%s" (include "fluxbase.fullname" .) -}}
{{- end }}
{{- end }}

{{/*
Return the proper PostgreSQL password secret key
*/}}
{{- define "fluxbase.databaseSecretPasswordKey" -}}
{{- if .Values.existingSecret }}
    {{- print "database-password" -}}
{{- else if eq .Values.postgresql.mode "standalone" }}
    {{- print "password" -}}
{{- else if eq .Values.postgresql.mode "cnpg" }}
    {{- print "password" -}}
{{- else if eq .Values.postgresql.mode "none" }}
    {{- if .Values.externalDatabase.existingSecret }}
        {{- .Values.externalDatabase.existingSecretPasswordKey -}}
    {{- else }}
        {{- print "database-password" -}}
    {{- end }}
{{- else }}
    {{- print "database-password" -}}
{{- end }}
{{- end }}

{{/*
Return the proper JWT secret name
*/}}
{{- define "fluxbase.jwtSecretName" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecret -}}
{{- else }}
    {{- printf "%s" (include "fluxbase.fullname" .) -}}
{{- end }}
{{- end }}

{{/*
Compile all warnings into a single message.
*/}}
{{- define "fluxbase.validateValues" -}}
{{- $messages := list -}}
{{- $messages := append $messages (include "fluxbase.validateValues.database" .) -}}
{{- $messages := append $messages (include "fluxbase.validateValues.jwt" .) -}}
{{- $messages := without $messages "" -}}
{{- $message := join "\n" $messages -}}

{{- if $message -}}
{{-   printf "\nVALUES VALIDATION:\n%s" $message -}}
{{- end -}}
{{- end -}}

{{/*
Validate database configuration
*/}}
{{- define "fluxbase.validateValues.database" -}}
{{- if and (not .Values.postgresql.enabled) (not .Values.externalDatabase.host) -}}
fluxbase: database
    You disabled the PostgreSQL sub-chart but did not specify an external PostgreSQL host.
    Please set postgresql.enabled=true or configure externalDatabase.host
{{- end -}}
{{- end -}}

{{/*
Validate JWT secret
*/}}
{{- define "fluxbase.validateValues.jwt" -}}
{{- if and (not .Values.existingSecret) (not .Values.fluxbase.auth.jwt_secret) -}}
fluxbase: auth.jwt_secret
    You must provide either existingSecret or fluxbase.auth.jwt_secret for JWT authentication.
    Please set one of these values or create a secret with key 'jwt-secret'
{{- end -}}
{{- end -}}

{{/*
Return true if cert-manager required annotations for TLS signed certificates are set in the Ingress annotations
Ref: https://cert-manager.io/docs/usage/ingress/#supported-annotations
*/}}
{{- define "fluxbase.ingress.certManagerRequest" -}}
{{ if or (hasKey . "cert-manager.io/cluster-issuer") (hasKey . "cert-manager.io/issuer") }}
    {{- true -}}
{{- end -}}
{{- end -}}

{{/*
Create a default fully qualified app name for PostgreSQL subchart
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "fluxbase.postgresql.fullname" -}}
{{- printf "%s-postgresql" (include "fluxbase.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return the resourcePreset value according to the values or defaults
*/}}
{{- define "fluxbase.resources" -}}
{{- if .Values.resources }}
{{- toYaml .Values.resources }}
{{- else }}
{{- $preset := .Values.resourcesPreset | default "small" }}
{{- if eq $preset "nano" }}
requests:
  cpu: 50m
  memory: 64Mi
limits:
  cpu: 100m
  memory: 128Mi
{{- else if eq $preset "micro" }}
requests:
  cpu: 100m
  memory: 128Mi
limits:
  cpu: 200m
  memory: 256Mi
{{- else if eq $preset "small" }}
requests:
  cpu: 250m
  memory: 256Mi
limits:
  cpu: 500m
  memory: 512Mi
{{- else if eq $preset "medium" }}
requests:
  cpu: 500m
  memory: 512Mi
limits:
  cpu: 1000m
  memory: 1Gi
{{- else if eq $preset "large" }}
requests:
  cpu: 1000m
  memory: 1Gi
limits:
  cpu: 2000m
  memory: 2Gi
{{- else if eq $preset "xlarge" }}
requests:
  cpu: 2000m
  memory: 2Gi
limits:
  cpu: 4000m
  memory: 4Gi
{{- else if eq $preset "2xlarge" }}
requests:
  cpu: 4000m
  memory: 4Gi
limits:
  cpu: 8000m
  memory: 8Gi
{{- end }}
{{- end }}
{{- end -}}
{{/*
Return the namespace
*/}}
{{- define "fluxbase.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return a soft nodeAffinity definition
*/}}
{{- define "common.affinities.nodes.soft" -}}
preferredDuringSchedulingIgnoredDuringExecution:
  - preference:
      matchExpressions:
        - key: {{ .key }}
          operator: In
          values:
            {{- range .values }}
            - {{ . | quote }}
            {{- end }}
    weight: 1
{{- end -}}

{{/*
Return a hard nodeAffinity definition
*/}}
{{- define "common.affinities.nodes.hard" -}}
requiredDuringSchedulingIgnoredDuringExecution:
  nodeSelectorTerms:
    - matchExpressions:
        - key: {{ .key }}
          operator: In
          values:
            {{- range .values }}
            - {{ . | quote }}
            {{- end }}
{{- end -}}

{{/*
Return a nodeAffinity definition
*/}}
{{- define "common.affinities.nodes" -}}
  {{- if .type -}}
    {{- if eq .type "soft" -}}
      {{- include "common.affinities.nodes.soft" . -}}
    {{- else if eq .type "hard" -}}
      {{- include "common.affinities.nodes.hard" . -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{/*
Return a soft podAffinity/podAntiAffinity definition
*/}}
{{- define "common.affinities.pods.soft" -}}
{{- $component := default "" .component -}}
preferredDuringSchedulingIgnoredDuringExecution:
  - podAffinityTerm:
      labelSelector:
        matchLabels: {{- (include "fluxbase.selectorLabels" .context) | nindent 10 }}
          {{- if not (empty $component) }}
          {{ printf "app.kubernetes.io/component: %s" $component }}
          {{- end }}
      topologyKey: kubernetes.io/hostname
    weight: 1
{{- end -}}

{{/*
Return a hard podAffinity/podAntiAffinity definition
*/}}
{{- define "common.affinities.pods.hard" -}}
{{- $component := default "" .component -}}
requiredDuringSchedulingIgnoredDuringExecution:
  - labelSelector:
      matchLabels: {{- (include "fluxbase.selectorLabels" .context) | nindent 8 }}
        {{- if not (empty $component) }}
        {{ printf "app.kubernetes.io/component: %s" $component }}
        {{- end }}
    topologyKey: kubernetes.io/hostname
{{- end -}}

{{/*
Return a podAffinity/podAntiAffinity definition
*/}}
{{- define "common.affinities.pods" -}}
  {{- if .type -}}
    {{- if eq .type "soft" -}}
      {{- include "common.affinities.pods.soft" . -}}
    {{- else if eq .type "hard" -}}
      {{- include "common.affinities.pods.hard" . -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
