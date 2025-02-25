module code.waarp.fr/apps/gateway/gateway

go 1.23

require (
	code.waarp.fr/lib/goftp v0.1.1-0.20240801120050-932aa2b2f9a7
	code.waarp.fr/lib/log v1.2.0
	code.waarp.fr/lib/migration v0.4.0
	code.waarp.fr/lib/r66 v0.1.6
	github.com/bwmarrin/snowflake v0.3.0
	github.com/dustin/go-humanize v1.0.1
	github.com/fclairamb/ftpserverlib v0.24.1
	github.com/fclairamb/go-log v0.5.0
	github.com/go-sql-driver/mysql v1.8.1
	github.com/google/uuid v1.6.0
	github.com/gookit/color v1.5.4
	github.com/gorilla/mux v1.8.1
	github.com/gosnmp/gosnmp v1.38.0
	github.com/indece-official/go-ebcdic v1.2.0
	github.com/jackc/pgerrcode v0.0.0-20240316143900-6e2875d9b438
	github.com/jackc/pgx/v5 v5.7.1
	github.com/jessevdk/go-flags v1.6.1
	github.com/karrick/tparse/v2 v2.8.2
	github.com/mattn/go-colorable v0.1.13
	github.com/pkg/sftp v1.13.6
	github.com/puzpuzpuz/xsync v1.5.2
	github.com/rclone/rclone v1.68.1
	github.com/slayercat/GoSNMPServer v0.5.2
	github.com/smartystreets/goconvey v1.8.1
	github.com/spf13/afero v1.11.0
	github.com/stretchr/testify v1.9.0
	golang.org/x/crypto v0.28.0
	golang.org/x/exp v0.0.0-20241009180824-f66d83c29e7c
	golang.org/x/net v0.30.0
	golang.org/x/term v0.25.0
	golang.org/x/text v0.19.0
	modernc.org/sqlite v1.33.1
	xorm.io/builder v0.3.13
	xorm.io/xorm v1.3.2 //do not update, newer versions break "SELECT FOR UPDATE" on sqlite & postgres
)

require (
	code.bcarlin.xyz/go/logging v0.4.1 // indirect
	code.waarp.fr/lib/Sqlite3CreateTableParser v0.0.0-20221122183218-24f478e49362 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Max-Sum/base32768 v0.0.0-20230304063302-18e6ce5945fd // indirect
	github.com/abbot/go-http-auth v0.4.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.30.3 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.3 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.27.27 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.27 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.11 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.3.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.17.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.58.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.22.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.26.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.30.3 // indirect
	github.com/aws/smithy-go v1.20.3 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-chi/chi/v5 v5.1.0 // indirect
	github.com/go-darwin/apfs v0.0.0-20211011131704-f84b94dbf348 // indirect
	github.com/go-git/go-billy/v5 v5.5.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/jzelinskie/whirlpool v0.0.0-20201016144138-0675e54bb004 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20231016141302-07b5767bb0ed // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/ncw/swift/v2 v2.0.3 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/xattr v0.4.9 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20221212215047-62379fc7944b // indirect
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rfjakob/eme v1.1.2 // indirect
	github.com/shirou/gopsutil/v3 v3.24.5 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/smarty/assertions v1.15.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/syndtr/goleveldb v1.0.0 // indirect
	github.com/tklauser/go-sysconf v0.3.13 // indirect
	github.com/tklauser/numcpus v0.7.0 // indirect
	github.com/xo/terminfo v0.0.0-20210125001918-ca9a967f8778 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/gc/v3 v3.0.0-20240107210532-573471604cb6 // indirect
	modernc.org/libc v1.55.3 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
)
