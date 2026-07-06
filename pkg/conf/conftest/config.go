package conftest

import (
	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// InitTestOverrides is a test helper function to quickly initiate the
// LocalOverrides global variable. This function should only be used in tests.
func InitTestOverrides(c convey.C, db *database.DB) {
	file := testhelpers.TempFile(c, "test_addr_override_*.ini")
	if db.Config == nil {
		db.Config = &conf.ServerConfig{}
	}

	db.Config.Overrides = conf.NewOverride(file)
	c.So(db.Config.Overrides.ListenAddresses.Parse(), convey.ShouldBeNil)
}
