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
{{- $termination := (.Values.image.tag | default .Chart.AppVersion) | toString -}}
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
{{- if .Values.config.database.host }}
    {{- .Values.config.database.host -}}
{{- else if .Values.postgresql.enabled }}
    {{- printf "%s-postgresql" (include "fluxbase.fullname" .) -}}
{{- else }}
    {{- .Values.externalDatabase.host -}}
{{- end }}
{{- end }}

{{/*
Return the proper PostgreSQL port
*/}}
{{- define "fluxbase.databasePort" -}}
{{- if .Values.postgresql.enabled }}
    {{- print "5432" -}}
{{- else }}
    {{- .Values.externalDatabase.port -}}
{{- end }}
{{- end }}

{{/*
Return the proper PostgreSQL database name
*/}}
{{- define "fluxbase.databaseName" -}}
{{- if .Values.postgresql.enabled }}
    {{- .Values.postgresql.auth.database -}}
{{- else }}
    {{- .Values.externalDatabase.database -}}
{{- end }}
{{- end }}

{{/*
Return the proper PostgreSQL runtime username
*/}}
{{- define "fluxbase.databaseUser" -}}
{{- if .Values.config.database.user }}
    {{- .Values.config.database.user -}}
{{- else if .Values.postgresql.enabled }}
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
{{- else if .Values.postgresql.enabled }}
    {{- if .Values.postgresql.auth.existingSecret }}
        {{- .Values.postgresql.auth.existingSecret -}}
    {{- else }}
        {{- printf "%s-postgresql" (include "fluxbase.fullname" .) -}}
    {{- end }}
{{- else if .Values.externalDatabase.existingSecret }}
    {{- .Values.externalDatabase.existingSecret -}}
{{- else }}
    {{- printf "%s" (include "fluxbase.fullname" .) -}}
{{- end }}
{{- end }}

{{/*
Return the proper PostgreSQL password secret key
*/}}
{{- define "fluxbase.databaseSecretPasswordKey" -}}
{{- if .Values.existingSecretKeyRef.databasePassword }}
    {{- .Values.existingSecretKeyRef.databasePassword -}}
{{- else if .Values.postgresql.enabled }}
    {{- .Values.postgresql.auth.secretKeys.password | default "db-password" -}}
{{- else if .Values.externalDatabase.existingSecret }}
    {{- .Values.externalDatabase.existingSecretPasswordKey -}}
{{- else }}
    {{- print "database-password" -}}
{{- end }}
{{- end }}

{{/*
Return the PostgreSQL admin username secret key
*/}}
{{- define "fluxbase.databaseSecretAdminUsernameKey" -}}
{{- if .Values.postgresql.enabled }}
    {{- .Values.postgresql.auth.secretKeys.adminUsername | default "db-admin-username" -}}
{{- else }}
    {{- print "db-admin-username" -}}
{{- end }}
{{- end }}

{{/*
Return the PostgreSQL admin password secret key
*/}}
{{- define "fluxbase.databaseSecretAdminPasswordKey" -}}
{{- if .Values.existingSecretKeyRef.databaseAdminPassword }}
    {{- .Values.existingSecretKeyRef.databaseAdminPassword -}}
{{- else if .Values.postgresql.enabled }}
    {{- .Values.postgresql.auth.secretKeys.adminPassword | default "db-admin-password" -}}
{{- else }}
    {{- print "database-admin-password" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for JWT secret
*/}}
{{- define "fluxbase.secretKeyRef.jwtSecret" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.jwtSecret | default "jwt-secret" -}}
{{- else }}
    {{- print "jwt-secret" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for anon key
*/}}
{{- define "fluxbase.secretKeyRef.anonKey" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.anonKey | default "anon-key" -}}
{{- else }}
    {{- print "anon-key" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for service role key
*/}}
{{- define "fluxbase.secretKeyRef.serviceRoleKey" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.serviceRoleKey | default "service-role-key" -}}
{{- else }}
    {{- print "service-role-key" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for setup token
*/}}
{{- define "fluxbase.secretKeyRef.setupToken" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.setupToken | default "setup-token" -}}
{{- else }}
    {{- print "setup-token" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for S3 access key
*/}}
{{- define "fluxbase.secretKeyRef.s3AccessKey" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.s3AccessKey | default "s3-access-key" -}}
{{- else }}
    {{- print "s3-access-key" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for S3 secret key
*/}}
{{- define "fluxbase.secretKeyRef.s3SecretKey" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.s3SecretKey | default "s3-secret-key" -}}
{{- else }}
    {{- print "s3-secret-key" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for SMTP password
*/}}
{{- define "fluxbase.secretKeyRef.smtpPassword" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.smtpPassword | default "smtp-password" -}}
{{- else }}
    {{- print "smtp-password" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for SendGrid API key
*/}}
{{- define "fluxbase.secretKeyRef.sendgridApiKey" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.sendgridApiKey | default "sendgrid-api-key" -}}
{{- else }}
    {{- print "sendgrid-api-key" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for Mailgun API key
*/}}
{{- define "fluxbase.secretKeyRef.mailgunApiKey" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.mailgunApiKey | default "mailgun-api-key" -}}
{{- else }}
    {{- print "mailgun-api-key" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for SES access key
*/}}
{{- define "fluxbase.secretKeyRef.sesAccessKey" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.sesAccessKey | default "ses-access-key" -}}
{{- else }}
    {{- print "ses-access-key" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for SES secret key
*/}}
{{- define "fluxbase.secretKeyRef.sesSecretKey" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.sesSecretKey | default "ses-secret-key" -}}
{{- else }}
    {{- print "ses-secret-key" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for AI encryption key
*/}}
{{- define "fluxbase.secretKeyRef.aiEncryptionKey" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.aiEncryptionKey | default "ai-encryption-key" -}}
{{- else }}
    {{- print "ai-encryption-key" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for OpenAI API key
*/}}
{{- define "fluxbase.secretKeyRef.openaiApiKey" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.openaiApiKey | default "openai-api-key" -}}
{{- else }}
    {{- print "openai-api-key" -}}
{{- end }}
{{- end }}

{{/*
Return the secret key for Azure API key
*/}}
{{- define "fluxbase.secretKeyRef.azureApiKey" -}}
{{- if .Values.existingSecret }}
    {{- .Values.existingSecretKeyRef.azureApiKey | default "azure-api-key" -}}
{{- else }}
    {{- print "azure-api-key" -}}
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
{{- if and (not .Values.existingSecret) (not .Values.config.auth.jwt_secret) -}}
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
