module code.waarp.fr/apps/gateway/gateway

go 1.20

require (
	code.waarp.fr/lib/goftp v0.1.1-0.20240410131823-5c6be29f291d
	code.waarp.fr/lib/log v1.2.0
	code.waarp.fr/lib/migration v0.4.0
	code.waarp.fr/lib/r66 v0.1.6
	github.com/bwmarrin/snowflake v0.3.0
	github.com/fclairamb/ftpserverlib v0.23.0 //do not update, uses go version >1.20
	github.com/fclairamb/go-log v0.4.1 //do not update, uses go version >1.20
	github.com/go-sql-driver/mysql v1.8.1
	github.com/google/uuid v1.6.0
	github.com/gookit/color v1.5.4
	github.com/gorilla/mux v1.8.1
	github.com/hack-pad/hackpadfs v0.2.1
	github.com/jackc/pgerrcode v0.0.0-20240316143900-6e2875d9b438
	github.com/jackc/pgx/v5 v5.5.5
	github.com/jedib0t/go-pretty/v6 v6.5.8
	github.com/jessevdk/go-flags v1.5.0
	github.com/karrick/tparse/v2 v2.8.2
	github.com/mattn/go-colorable v0.1.13
	github.com/pkg/sftp v1.13.6
	github.com/puzpuzpuz/xsync v1.5.2
	github.com/smarty/assertions v1.15.1
	github.com/smartystreets/goconvey v1.8.1
	github.com/spf13/afero v1.11.0
	github.com/stretchr/testify v1.9.0
	go.step.sm/crypto v0.44.3
	golang.org/x/crypto v0.22.0
	golang.org/x/exp v0.0.0-20240409090435-93d18d7e34b8
	golang.org/x/net v0.24.0
	golang.org/x/term v0.19.0
	modernc.org/sqlite v1.29.6
	xorm.io/builder v0.3.13
	xorm.io/xorm v1.3.2 //do not update, newer versions break "SELECT FOR UPDATE" on sqlite & postgres
)

require (
	code.bcarlin.xyz/go/logging v0.4.1 // indirect
	code.waarp.fr/lib/Sqlite3CreateTableParser v0.0.0-20221122183218-24f478e49362 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/goccy/go-json v0.8.1 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/syndtr/goleveldb v1.0.0 // indirect
	github.com/xo/terminfo v0.0.0-20210125001918-ca9a967f8778 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/gc/v3 v3.0.0-20240107210532-573471604cb6 // indirect
	modernc.org/libc v1.41.0 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.7.2 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
)
