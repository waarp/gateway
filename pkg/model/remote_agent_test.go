package model

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestRemoteAgentTableName(t *testing.T) {
	Convey("Given a `RemoteAgent` instance", t, func() {
		agent := &RemoteAgent{}

		Convey("When calling the 'TableName' method", func() {
			name := agent.TableName()

			Convey("Then it should return the name of the remote agents table", func() {
				So(name, ShouldEqual, TableRemAgents)
			})
		})
	})
}

func TestRemoteAgentBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a remote agent entry", func() {
			ag := RemoteAgent{
				Name: "partner", Protocol: testProtocol,
				Address: types.Addr("localhost", 6666),
			}
			So(db.Insert(&ag).Run(), ShouldBeNil)

			acc := RemoteAccount{RemoteAgentID: ag.ID, Login: "foo"}
			So(db.Insert(&acc).Run(), ShouldBeNil)

			rule := Rule{Name: "rule", IsSend: false, Path: "path"}
			So(db.Insert(&rule).Run(), ShouldBeNil)

			agAccess := RuleAccess{
				RuleID:        rule.ID,
				RemoteAgentID: utils.NewNullInt64(ag.ID),
			}
			So(db.Insert(&agAccess).Run(), ShouldBeNil)

			accAccess := RuleAccess{
				RuleID:          rule.ID,
				RemoteAccountID: utils.NewNullInt64(acc.ID),
			}
			So(db.Insert(&accAccess).Run(), ShouldBeNil)

			authAg := Credential{
				RemoteAgentID: utils.NewNullInt64(ag.ID),
				Name:          "test agent cert",
				Type:          testInternalAuth,
				Value:         "val",
			}
			So(db.Insert(&authAg).Run(), ShouldBeNil)

			authAcc := Credential{
				RemoteAccountID: utils.NewNullInt64(acc.ID),
				Name:            "test account cert",
				Type:            testExternalAuth,
				Value:           "val",
			}
			So(db.Insert(&authAcc).Run(), ShouldBeNil)

			Convey("Given that the agent is unused", func() {
				Convey("When calling the `BeforeDelete` hook", func() {
					So(ag.BeforeDelete(db), ShouldBeNil)
				})

				Convey("When deleting the agent", func() {
					So(db.Delete(&ag).Run(), ShouldBeNil)

					Convey("Then the agent should have been deleted", func() {
						var agents RemoteAgents
						So(db.Select(&agents).Run(), ShouldBeNil)
						So(agents, ShouldBeEmpty)
					})

					Convey("Then the agent's accounts should have been deleted", func() {
						var accounts RemoteAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(accounts, ShouldBeEmpty)
					})

					Convey("Then both auth methods should have been deleted", func() {
						var auths Credentials
						So(db.Select(&auths).Run(), ShouldBeNil)
						So(auths, ShouldBeEmpty)
					})

					Convey("Then the rule accesses should have been deleted", func() {
						var perms RuleAccesses
						So(db.Select(&perms).Run(), ShouldBeNil)
						So(perms, ShouldBeEmpty)
					})
				})
			})

			Convey("Given that the agent is used in a transfer", func() {
				cli := &Client{Protocol: ag.Protocol}
				So(db.Insert(cli).Run(), ShouldBeNil)

				trans := &Transfer{
					RuleID:          rule.ID,
					ClientID:        utils.NewNullInt64(cli.ID),
					RemoteAccountID: utils.NewNullInt64(acc.ID),
					SrcFilename:     "file",
				}
				So(db.Insert(trans).Run(), ShouldBeNil)

				Convey("When calling the `BeforeDelete` hook", func() {
					err := ag.BeforeDelete(db)

					Convey("Then it should say that the agent is being used", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"this partner is currently being used in one or more "+
								"running transfers and thus cannot be deleted, "+
								"cancel these transfers or wait for them to finish"))
					})
				})
			})
		})
	})
}

func TestRemoteAgentValidate(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given the database contains 1 remote agent", func() {
			oldAgent := RemoteAgent{
				Name: "old", Protocol: testProtocol,
				Address: types.Addr("localhost", 2022),
			}
			So(db.Insert(&oldAgent).Run(), ShouldBeNil)

			Convey("Given a new remote agent", func() {
				newAgent := &RemoteAgent{
					Name: "new", Protocol: testProtocol,
					Address: types.Addr("localhost", 2023),
				}

				shouldFailWith := func(expMsg string, args ...any) {
					expErr := fmt.Sprintf(expMsg, args...)

					Convey("When calling the 'BeforeWrite' function", func() {
						err := db.Transaction(func(ses *database.Session) error {
							return newAgent.BeforeWrite(ses)
						})

						Convey("Then the error should say that "+expErr, func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldContainSubstring, expErr)
						})
					})
				}

				Convey("Given that the new agent is valid", func() {
					Convey("When calling the 'BeforeWrite' function", func() {
						err := db.Transaction(func(ses *database.Session) error {
							return newAgent.BeforeWrite(ses)
						})

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new agent is missing a name", func() {
					newAgent.Name = ""

					shouldFailWith("the agent's name cannot be empty")
				})

				Convey("Given that the new agent's name is already taken", func() {
					newAgent.Name = oldAgent.Name

					shouldFailWith(
						"a remote agent with the same name %q already exist",
						newAgent.Name)
				})

				Convey("Given that the new agent is missing an address", func() {
					newAgent.Address = types.Address{}

					shouldFailWith("%s", types.ErrEmptyAddress.Error())
				})

				Convey("Given that the new agent's address is invalid", func() {
					newAgent.Address.Host = "not_an_address"

					shouldFailWith(`address validation failed`)
				})

				Convey("Given that the new agent's protocol is not valid", func() {
					newAgent.Protocol = "not a protocol"

					shouldFailWith(`unknown protocol "not a protocol"`)
				})
			})
		})
	})
}

func TestRemoteAgentAfterUpdate(t *testing.T) {
	db := dbtest.TestDatabase(t)
	partner := RemoteAgent{
		Owner:       "test_gateway",
		Name:        "new",
		Protocol:    "r66",
		ProtoConfig: map[string]any{},
		Address:     types.Addr("localhost", 2023),
	}
	require.NoError(t, db.Insert(&partner).Run())

	cleanup := func() {
		require.NoError(t, db.DeleteAll(&Credential{}).Run())
	}

	t.Run("No changes", func(t *testing.T) {
		t.Cleanup(cleanup)

		require.NoError(t, partner.AfterUpdate(db))
	})

	t.Run("New R66 server password", func(t *testing.T) {
		t.Cleanup(cleanup)

		const password = "sesame_hash"
		partner.ProtoConfig["serverPassword"] = password

		require.NoError(t, partner.AfterUpdate(db))

		var pswd Credential
		require.NoError(t, db.Get(&pswd, "remote_agent_id=? AND type=?",
			partner.ID, authPassword).Run())
		assert.Equal(t, password, pswd.Value)
	})

	t.Run("Existing R66 server password", func(t *testing.T) {
		t.Cleanup(cleanup)

		const (
			oldPassword = "sesame_hash"
			newPassword = "sesame2_hash"
		)

		require.NoError(t, db.Insert(&Credential{
			RemoteAgentID: utils.NewNullInt64(partner.ID),
			Name:          authPassword,
			Type:          authPassword,
			Value:         oldPassword,
		}).Run())

		partner.ProtoConfig["serverPassword"] = newPassword
		require.NoError(t, partner.AfterUpdate(db))

		var pswd Credential
		require.NoError(t, db.Get(&pswd, "remote_agent_id=? AND type=?",
			partner.ID, authPassword).Run())
		assert.Equal(t, newPassword, pswd.Value)
	})
}
