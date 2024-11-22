module github.com/soedev/soelib

go 1.22.7

require (
	github.com/Lofanmi/pinyin-golang v0.0.0-20211114132645-1db892057f20
	github.com/Shopify/sarama v1.19.0
	github.com/afex/hystrix-go v0.0.0-20180502004556-fa1af6a1f4f5
	github.com/alex023/clock v0.0.0-20191208111215-c265f1b2ab18
	github.com/astaxie/beego v1.12.1
	github.com/axgle/mahonia v0.0.0-20180208002826-3358181d7394
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/digitalocean/go-smbios v0.0.0-20180907143718-390a4f403a8e
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/getsentry/raven-go v0.2.0
	github.com/getsentry/sentry-go v0.9.0
	github.com/gin-gonic/gin v1.9.1
	github.com/go-ini/ini v1.54.0
	github.com/golang/protobuf v1.5.4
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/google/uuid v1.6.0
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/mitchellh/mapstructure v1.5.0
	github.com/nacos-group/nacos-sdk-go v1.0.8
	github.com/opentracing/opentracing-go v1.1.0
	github.com/openzipkin/zipkin-go v0.2.2
	github.com/pkg/errors v0.9.1
	github.com/spf13/cast v1.3.0
	github.com/spf13/viper v1.6.2
	github.com/streadway/amqp v0.0.0-20190404075320-75d898a42a94
	github.com/uber/jaeger-client-go v2.23.1+incompatible
	github.com/ulule/deepcopier v0.0.0-20200117111125-792cfb847af8
	go.uber.org/zap v1.15.0
	golang.org/x/crypto v0.28.0
	golang.org/x/net v0.30.0
	golang.org/x/text v0.19.0
	google.golang.org/grpc v1.68.0
	google.golang.org/protobuf v1.35.2
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22
	gorm.io/driver/mysql v1.5.1
	gorm.io/driver/postgres v1.5.2
	gorm.io/driver/sqlserver v1.5.1
	gorm.io/gorm v1.25.1
)

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20241118233622-e639e219e697

// 使用本地sqlserver驱动代码： 本地代码自定义修改过，为了解决CREATE生成脚本问题
replace gorm.io/driver/sqlserver => ./pkg/sqlserver

require google.golang.org/genproto/googleapis/rpc v0.0.0-20241118233622-e639e219e697 // indirect

require (
	github.com/HuKeping/rbtree v0.0.0-20200208030951-29f0b79e84ed // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.18 // indirect
	github.com/buger/jsonparser v0.0.0-20181115193947-bf1c66bbce23 // indirect
	github.com/bytedance/sonic v1.9.1 // indirect
	github.com/certifi/gocertifi v0.0.0-20200211180108-c7c1fbc02894 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/eapache/go-resiliency v1.1.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.4.7 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.14.0 // indirect
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.5.4 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/lestrrat/go-file-rotatelogs v0.0.0-20180223000712-d3151e2a480f // indirect
	github.com/lestrrat/go-strftime v0.0.0-20180220042222-ba3bf9c1d042 // indirect
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/microsoft/go-mssqldb v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml v1.2.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/pierrec/lz4 v1.0.2-0.20190131084431-473cd7ce01a1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/spf13/afero v1.10.0 // indirect
	github.com/spf13/jwalterweatherman v1.0.0 // indirect
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/toolkits/concurrent v0.0.0-20150624120057-a4371d70e3e3 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/uber/jaeger-lib v2.2.0+incompatible // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	go.uber.org/atomic v1.6.0 // indirect
	go.uber.org/multierr v1.5.0 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	gopkg.in/ini.v1 v1.51.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
