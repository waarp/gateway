package migrations

import "fmt"

type Binary int64

func (b Binary) CanBeAutoIncr() bool    { return false }
func (b Binary) ToSqliteType() string   { return Blob{}.ToSqliteType() }
func (b Binary) ToPostgresType() string { return Blob{}.ToPostgresType() }
func (b Binary) ToMysqlType() string    { return fmt.Sprintf("BINARY(%d)", b) }

type UnsignedBigInt struct{}

func (UnsignedBigInt) CanBeAutoIncr() bool    { return true }
func (UnsignedBigInt) ToSqliteType() string   { return BigInt{}.ToSqliteType() }
func (UnsignedBigInt) ToPostgresType() string { return BigInt{}.ToPostgresType() }
func (UnsignedBigInt) ToMysqlType() string    { return "BIGINT UNSIGNED" }
