{{/*
Default basic database configuration, handling external database scenarios
*/}}
{{- define "bkms-server.database" -}}
{{- $root := first . -}}
{{- $name := last . -}}
{{- $values := $root.Values -}}
{{- $db := index $values.externalDatabase $name -}}
{{- $dbDefault := $values.externalDatabase.default -}}
name: {{ $db.name | default $dbDefault.name | required (printf "externalDatabase.%s.name is required" $name) }}
user: {{ $db.user | default $dbDefault.user | required (printf "externalDatabase.%s.user is required" $name) }}
password: {{ $db.password | default $dbDefault.password | required (printf "externalDatabase.%s.password is required" $name) }}
host: {{ $db.host | default $dbDefault.host | required (printf "externalDatabase.%s.host is required" $name) }}
port: {{ $db.port | default $dbDefault.port | required (printf "externalDatabase.%s.port is required" $name) }}
{{- end -}}

{{- define "bkms-server.database.default" -}}
{{ include "bkms-server.database" (list . "default") }}
{{- end -}}

{{/*
Default basic redis configuration, handling external redis scenarios
*/}}
{{- define "bkms-server.redis" -}}
{{- $root := first . -}}
{{- $name := last . -}}
{{- $values := $root.Values -}}
{{- $redis := index $values.externalRedis $name -}}
{{- $redisDefault := $values.externalRedis.default -}}
host: {{ $redis.host | default $redisDefault.host | required (printf "externalRedis.%s.host is required" $name) }}
port: {{ $redis.port | default $redisDefault.port | required (printf "externalRedis.%s.port is required" $name) }}
db: {{ $redis.db | default $redisDefault.db | required (printf "externalRedis.%s.db is required" $name) }}
password: {{ $redis.password | default $redisDefault.password | required (printf "externalRedis.%s.password is required" $name) }}
dialTimeout: {{ $redis.dialTimeout | default $redisDefault.dialTimeout | required (printf "externalRedis.%s.dialTimeout is required" $name) }}
readTimeout: {{ $redis.readTimeout | default $redisDefault.readTimeout | required (printf "externalRedis.%s.readTimeout is required" $name) }}
writeTimeout: {{ $redis.writeTimeout | default $redisDefault.writeTimeout | required (printf "externalRedis.%s.writeTimeout is required" $name) }}
{{- end -}}

{{- define "bkms-server.redis.default" -}}
{{ include "bkms-server.redis" (list . "default") }}
{{- end -}}
