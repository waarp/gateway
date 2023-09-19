package model

import (
	"errors"
	"fmt"

	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

const (
	testProtocol        = "test_proto"
	testProtocolInvalid = "test_proto_invalid"

	testLocalPath = "file:/test/local/file"
)

var (
	errInvalidProtoConfig = errors.New("invalid protocol configuration")
	errUnknownProtocol    = errors.New("unknown protocol")
)

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	ConfigChecker = testConfigChecker{
		testProtocol:        nil,
		testProtocolInvalid: errInvalidProtoConfig,
	}
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)

	return string(h)
}

func mkURL(str string) types.URL {
	url, err := types.ParseURL(str)
	convey.So(err, convey.ShouldBeNil)

	return *url
}

type testConfigChecker map[string]error

func (t testConfigChecker) checkConfig(proto string) error {
	if err, ok := t[proto]; ok {
		return err
	}

	return fmt.Errorf("%w %q", errUnknownProtocol, proto)
}

func (t testConfigChecker) IsValidProtocol(proto string) bool {
	return !errors.Is(t.checkConfig(proto), errUnknownProtocol)
}

func (t testConfigChecker) CheckServerConfig(proto string, _ map[string]any) error {
	return t.checkConfig(proto)
}

func (t testConfigChecker) CheckClientConfig(proto string, _ map[string]any) error {
	return t.checkConfig(proto)
}

func (t testConfigChecker) CheckPartnerConfig(proto string, _ map[string]any) error {
	return t.checkConfig(proto)
}
