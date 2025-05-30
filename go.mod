module code.waarp.fr/apps/gateway/gateway

go 1.23.8

toolchain go1.24.3

require (
	code.waarp.fr/lib/goftp v0.1.1-0.20240801120050-932aa2b2f9a7
	code.waarp.fr/lib/log v1.3.1
	code.waarp.fr/lib/migration v0.4.0
	code.waarp.fr/lib/pesit v0.1.2
	code.waarp.fr/lib/r66 v0.1.7
	github.com/ProtonMail/gopenpgp/v3 v3.3.0
	github.com/bwmarrin/snowflake v0.3.0
	github.com/dsnet/compress v0.0.1
	github.com/dustin/go-humanize v1.0.1
	github.com/egirna/icap v0.3.2
	github.com/fclairamb/ftpserverlib v0.25.0
	github.com/fclairamb/go-log v0.5.0
	github.com/go-sql-driver/mysql v1.9.2
	github.com/google/uuid v1.6.0
	github.com/gookit/color v1.5.4
	github.com/gorilla/mux v1.8.1
	github.com/gosnmp/gosnmp v1.40.0
	github.com/indece-official/go-ebcdic v1.2.0
	github.com/jackc/pgerrcode v0.0.0-20240316143900-6e2875d9b438
	github.com/jackc/pgx/v5 v5.7.5
	github.com/jessevdk/go-flags v1.6.1
	github.com/karrick/tparse/v2 v2.8.2
	github.com/klauspost/compress v1.18.0
	github.com/mattn/go-colorable v0.1.14
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58
	github.com/pkg/sftp v1.13.9
	github.com/puzpuzpuz/xsync v1.5.2
	github.com/rclone/rclone v1.69.3
	github.com/slayercat/GoSNMPServer v0.5.2
	github.com/smartystreets/goconvey v1.8.1
	github.com/solidwall/icap-client v0.0.0-20240910140159-0c244324de0d
	github.com/spf13/afero v1.14.0
	github.com/stretchr/testify v1.10.0
	github.com/ulikunitz/xz v0.5.12
	golang.org/x/crypto v0.38.0
	golang.org/x/exp v0.0.0-20250506013437-ce4c2cf36ca6
	golang.org/x/net v0.40.0
	golang.org/x/term v0.32.0
	golang.org/x/text v0.25.0
	modernc.org/sqlite v1.37.1
	xorm.io/builder v0.3.13
	xorm.io/xorm v1.3.2 //do not update, newer versions break "SELECT FOR UPDATE" on sqlite & postgres
)

require (
	code.bcarlin.net/go/logging v0.5.1 // indirect
	code.waarp.fr/lib/Sqlite3CreateTableParser v0.0.0-20221122183218-24f478e49362 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.18.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.10.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.6.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azfile v1.5.1 // indirect
	github.com/Files-com/files-sdk-go/v3 v3.2.169 // indirect
	github.com/Max-Sum/base32768 v0.0.0-20230304063302-18e6ce5945fd // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.3.0 // indirect
	github.com/ProtonMail/gopenpgp/v2 v2.9.0 // indirect
	github.com/PuerkitoBio/goquery v1.10.3 // indirect
	github.com/abbot/go-http-auth v0.4.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.36.3 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.10 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.29.14 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.67 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.77 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.7.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.80.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.19 // indirect
	github.com/aws/smithy-go v1.22.3 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bradenaw/juniper v0.15.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/cloudinary/cloudinary-go/v2 v2.10.0 // indirect
	github.com/cloudsoda/go-smb2 v0.0.0-20250228001242-d4c70e6251cc // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/creasty/defaults v1.8.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/ebitengine/purego v0.8.4 // indirect
	github.com/emersion/go-message v0.18.2 // indirect
	github.com/emersion/go-vcard v0.0.0-20241024213814-c9703dde27ff // indirect
	github.com/flynn/noise v1.1.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.9 // indirect
	github.com/go-chi/chi/v5 v5.2.1 // indirect
	github.com/go-darwin/apfs v0.0.0-20211011131704-f84b94dbf348 // indirect
	github.com/go-git/go-billy/v5 v5.6.2 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-resty/resty/v2 v2.16.5 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/josephu/go v0.1.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/jzelinskie/whirlpool v0.0.0-20201016144138-0675e54bb004 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20250317134145-8bc96cf8fc35 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.17 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/ncw/swift/v2 v2.0.3 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/oracle/oci-go-sdk/v65 v65.92.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/xattr v0.4.10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.64.0 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/relvacode/iso8601 v1.6.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rfjakob/eme v1.1.2 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/shirou/gopsutil/v3 v3.24.5 // indirect
	github.com/shirou/gopsutil/v4 v4.25.4 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/smarty/assertions v1.15.0 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/spacemonkeygo/monkit/v3 v3.0.24 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/syndtr/goleveldb v1.0.0 // indirect
	github.com/t3rm1n4l/go-mega v0.0.0-20241213151442-a19cff0ec7b5 // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zeebo/blake3 v0.2.4 // indirect
	go.etcd.io/bbolt v1.4.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	google.golang.org/api v0.235.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	google.golang.org/grpc v1.72.2 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.65.8 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	storj.io/common v0.0.0-20250529081337-15d9add689d4 // indirect
	storj.io/eventkit v0.0.0-20250410172343-61f26d3de156 // indirect
)
