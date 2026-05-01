module github.com/nimbleflux/fluxbase

go 1.26.1

require (
	github.com/ClickHouse/clickhouse-go/v2 v2.44.0
	github.com/aws/aws-sdk-go-v2 v1.41.5
	github.com/aws/aws-sdk-go-v2/credentials v1.19.14
	github.com/aws/aws-sdk-go-v2/service/ses v1.34.22
	github.com/beevik/etree v1.6.0
	github.com/coreos/go-oidc/v3 v3.17.0
	github.com/crewjam/saml v0.5.1
	github.com/davidbyttow/govips/v2 v2.18.0
	github.com/elastic/go-elasticsearch/v8 v8.19.3
	github.com/elastic/go-elasticsearch/v9 v9.3.1
	github.com/evanw/esbuild v0.28.0
	github.com/fsnotify/fsnotify v1.9.0
	github.com/go-viper/mapstructure/v2 v2.5.0
	github.com/gofiber/contrib/v3/websocket v1.1.0
	github.com/gofiber/fiber/v3 v3.2.0
	github.com/gofiber/storage/memory/v2 v2.1.1
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/graphql-go/graphql v0.8.1
	github.com/jackc/pgx/v5 v5.9.2
	github.com/joho/godotenv v1.5.1
	github.com/kapmahc/epub v0.1.1
	github.com/ledongthuc/pdf v0.0.0-20250511090121-5959a4027728
	github.com/mailgun/mailgun-go/v5 v5.15.0
	github.com/minio/minio-go/v7 v7.0.100
	github.com/nguyenthenguyen/docx v0.0.0-20230621112118-9c8e795a11db
	github.com/olekukonko/tablewriter v1.1.4
	github.com/otiai10/gosseract/v2 v2.4.1
	github.com/pganalyze/pg_query_go/v6 v6.2.2
	github.com/pgplex/pgparser v0.2.0
	github.com/pquerna/otp v1.5.0
	github.com/prometheus/client_golang v1.23.2
	github.com/redis/go-redis/v9 v9.18.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/zerolog v1.35.0
	github.com/russellhaering/goxmldsig v1.6.0
	github.com/sendgrid/sendgrid-go v3.16.1+incompatible
	github.com/shirou/gopsutil/v4 v4.26.3
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/spf13/cobra v1.10.2
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	github.com/twpayne/go-geom v1.6.1
	github.com/valyala/fasthttp v1.70.0
	github.com/xuri/excelize/v2 v2.10.1
	github.com/zalando/go-keyring v0.2.8
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.0
	go.opentelemetry.io/otel/sdk v1.43.0
	go.opentelemetry.io/otel/trace v1.43.0
	golang.org/x/crypto v0.50.0
	golang.org/x/oauth2 v0.36.0
	golang.org/x/term v0.42.0
	golang.org/x/time v0.15.0
	google.golang.org/grpc v1.80.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/ClickHouse/ch-go v0.71.0 // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.21 // indirect
	github.com/aws/smithy-go v1.24.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clipperhouse/displaywidth v0.10.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.6.0 // indirect
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.8.0 // indirect
	github.com/fasthttp/websocket v1.5.12 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/gofiber/schema v1.7.1 // indirect
	github.com/gofiber/utils/v2 v2.0.4 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/klauspost/cpuid/v2 v2.2.11 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.21 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oapi-codegen/runtime v1.3.1 // indirect
	github.com/olekukonko/cat v0.0.0-20250911104152-50322a0618f6 // indirect
	github.com/olekukonko/errors v1.2.0 // indirect
	github.com/olekukonko/ll v0.1.6 // indirect
	github.com/paulmach/orb v0.12.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.25 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/richardlehane/mscfb v1.0.6 // indirect
	github.com/richardlehane/msoleps v1.0.6 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/savsgio/gotils v0.0.0-20250924091648-bce9a52d7761 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/sendgrid/rest v2.6.9+incompatible // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tiendc/go-deepcopy v1.7.2 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/xuri/efp v0.0.1 // indirect
	github.com/xuri/nfp v0.0.2-0.20250530014748-2ddeb826f9a9 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/image v0.38.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
