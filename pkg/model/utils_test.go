package model

import (
	"errors"
	"fmt"

	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

const (
	testProtocol        = "test_proto"
	testProtocolInvalid = "test_proto_invalid"
	testInternalAuth    = "test_internal_auth"
	testExternalAuth    = "test_external_auth"
	testAuthority       = "test_authority"

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

	authentication.AddInternalCredentialType(testInternalAuth, &intAuth{})
	authentication.AddExternalCredentialType(testExternalAuth, &extAuth{})
	authentication.AddAuthorityType(testAuthority, &testAuthorityHandler{})
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
	_, ok := t[proto]

	return ok
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

type intAuth struct{}

func (*intAuth) CanOnlyHaveOne() bool                        { return false }
func (*intAuth) Validate(string, string, string, bool) error { return nil }
func (*intAuth) Authenticate(database.ReadAccess, authentication.Owner, any,
) (*authentication.Result, error) {
	return authentication.Success(), nil
}

type extAuth struct{}

func (*extAuth) CanOnlyHaveOne() bool                        { return false }
func (*extAuth) Validate(string, string, string, bool) error { return nil }

const invalidAuthorityVal = "authority public identity"

var errInvalidAuthorityVal = fmt.Errorf("%q is not a valid authority identity value",
	invalidAuthorityVal)

type testAuthorityHandler struct{}

func (*testAuthorityHandler) Validate(val string) error {
	if val != invalidAuthorityVal {
		return nil
	}

	return errInvalidAuthorityVal
}
