package model

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
)

const (
	testProtocol        = "test_proto"
	testProtocolInvalid = "test_proto_invalid"
	testInternalAuth    = "test_internal_auth"
	testExternalAuth    = "test_external_auth"
	testAuthority       = "test_authority"

	testLocalPath = "/test/local/file"
)

var errInvalidProtoConfig = errors.New("invalid protocol configuration")

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	Protocols[testProtocol] = dummyProtocol{}
	Protocols[testProtocolInvalid] = dummyProtocol{err: errInvalidProtoConfig}
	Protocols[protoR66] = dummyProtocol{}

	authentication.AddInternalCredentialType(testInternalAuth, &intAuth{})
	authentication.AddExternalCredentialType(testExternalAuth, &extAuth{})
	authentication.AddAuthorityType(testAuthority, &testAuthorityHandler{})

	authentication.AddInternalCredentialTypeForProtocol(authPassword, protoR66, &intAuth{})
	authentication.AddExternalCredentialTypeForProtocol(authPassword, protoR66, &extAuth{})
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)

	return string(h)
}

func localPath(fPath string) string {
	if runtime.GOOS == "windows" {
		fPath = "C:" + fPath
	}

	return fPath
}

type dummyProtocol struct{ err error }

func (dummyProtocol) CanMakeTransfer(*TransferContext) error    { return nil }
func (d dummyProtocol) CheckServerConfig(map[string]any) error  { return d.err }
func (d dummyProtocol) CheckClientConfig(map[string]any) error  { return d.err }
func (d dummyProtocol) CheckPartnerConfig(map[string]any) error { return d.err }

type intAuth struct{}

func (*intAuth) CanOnlyHaveOne() bool                                { return false }
func (*intAuth) Validate(string, string, string, string, bool) error { return nil }
func (*intAuth) Authenticate(database.ReadAccess, authentication.Owner, any,
) (*authentication.Result, error) {
	return authentication.Success(), nil
}

type extAuth struct{}

func (*extAuth) CanOnlyHaveOne() bool                                { return false }
func (*extAuth) Validate(string, string, string, string, bool) error { return nil }

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
