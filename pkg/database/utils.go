package database

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	vers "code.waarp.fr/apps/gateway/gateway/pkg/version"
)

//nolint:gochecknoinits //we want to disable GORM's default logger
func init() {
	logger.Default = logger.Discard
}

type badVersionError struct {
	exe, db string
}

func (e *badVersionError) Error() string {
	return fmt.Sprintf("database version mismatch (exe %q, db %q)", e.exe, e.db)
}

type exister interface {
	Table
	Identifier
}

func (db *DB) checkVersion() error {
	dbVer := &version{}

	if !db.engine.Migrator().HasTable(&dbVer) {
		return nil
	}

	if err := db.Get(dbVer, "").Run(); err != nil {
		db.Logger.Errorf("Failed to retrieve database version: %v", err)

		return err
	}

	if dbVer.Current != vers.Num {
		db.Logger.Criticalf("Mismatch between database (%s) and program (%s) versions.",
			dbVer.Current, vers.Num)

		return &badVersionError{exe: vers.Num, db: dbVer.Current}
	}

	return nil
}

func (s *Session) addOwner(bean any) {
	val := reflect.ValueOf(bean)
	for val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	if ownerField := val.FieldByName("Owner"); ownerField.IsValid() {
		ownerField.SetString(s.getOwner())
	}
}

func addOwnerCond(query *gorm.DB, all bool, bean any, owner string) {
	if all {
		return
	}

	typ := reflect.TypeOf(bean)
	for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return
	}

	if _, hasOwner := typ.FieldByName("Owner"); hasOwner {
		query.Where("owner=?", owner)
	}
}

func checkExists(db Access, bean exister) error {
	var n int64
	if err := db.getUnderlying().Table(bean.TableName()).Where("id=?", bean.GetID()).Count(&n).Error; err != nil {
		db.getLogger().Errorf("Failed to count bean: %v", err)

		return NewInternalError(err)
	}

	if n == 0 {
		db.getLogger().Debugf("No %s found with ID %d", bean.Appellation(), bean.GetID())

		return NewNotFoundError(bean)
	}

	return nil
}

func explainStmt(query *gorm.DB) string {
	return query.Statement.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx
	})
}

type gormLogger struct {
	*log.Logger
}

func (g *gormLogger) LogMode(logger.LogLevel) logger.Interface { return g }

func (g *gormLogger) Info(_ context.Context, s string, i ...any) {
	g.Infof(s, i...)
}

func (g *gormLogger) Warn(_ context.Context, s string, i ...any) {
	// Suppress warnings about hook interface.
	if strings.Contains(s, "Please see https://gorm.io/docs/hooks.html") {
		return
	}

	g.Warningf(s, i...)
}

func (g *gormLogger) Error(_ context.Context, s string, i ...any) {
	g.Errorf(s, i...)
}

func (g *gormLogger) Trace(_ context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		g.Errorf("%s - [%s] - %v", sql, elapsed, err)
	} else {
		g.Tracef("%s - [%s] - %d rows affected", sql, elapsed, rows)
	}
}

func AESEncrypt(gcm cipher.AEAD, password string) (string, error) {
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("cannot get random bytes: %w", err)
	}

	cipherBytes := gcm.Seal(nonce, nonce, []byte(password), nil)
	cipherText := base64.StdEncoding.EncodeToString(cipherBytes)

	return cipherText, nil
}

var errNonceTooLong = errors.New("the nonce cannot be longer than the text")

func AESDecrypt(gcm cipher.AEAD, cipherStr string) (string, error) {
	cryptPassword, err := base64.StdEncoding.DecodeString(cipherStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted password string: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(cryptPassword) < nonceSize {
		return "", errNonceTooLong
	}

	nonce, cipherText := cryptPassword[:nonceSize], cryptPassword[nonceSize:]

	password, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", fmt.Errorf("cannot decrypt password: %w", err)
	}

	return string(password), nil
}

func addPreloads(eager bool, query *gorm.DB, bean any) {
	if !eager {
		return
	}

	if preloader, ok := bean.(Preloader); ok {
		for _, preload := range preloader.Preloads() {
			query.Preload(preload)
		}
	}
}

func makeInClause(col string, vals ...any) *condition {
	if len(vals) == 0 {
		return &condition{}
	}

	if len(vals) == 1 {
		val := vals[0]
		if reflect.TypeOf(val).Kind() == reflect.Slice {
			return &condition{
				sql:  fmt.Sprintf("%s IN ?", col),
				args: []any{val},
			}
		}
	}

	return &condition{
		sql:  fmt.Sprintf("%s IN ?", col),
		args: []any{vals},
	}
}
