package wg

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func certInfoString(c *api.OutCrypto) string {
	return "● Certificate " + c.Name + "\n" +
		"    Private key: " + c.PrivateKey + "\n" +
		"    Public key:  " + c.PublicKey + "\n" +
		"    Content:     " + c.Certificate + "\n"
}

func resetVars() {
	Server = ""
	Partner = ""
	LocalAccount = ""
	RemoteAccount = ""
}

//nolint:maintidx //FIXME factorize the function if possible to improve maintainability
func TestGetCertificate(t *testing.T) {
	Convey("Testing the certificate 'get' command", t, func() {
		resetVars()
		out = testFile()
		command := &CertGet{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a partner", func() {
				partner := &model.RemoteAgent{
					Name:     "partner",
					Protocol: testProto1,
					Address:  "localhost:6666",
				}
				So(db.Insert(partner).Run(), ShouldBeNil)

				Convey("Given a partner certificate", func() {
					cert := &model.Crypto{
						RemoteAgentID: utils.NewNullInt64(partner.ID),
						Name:          "partner_cert",
						Certificate:   testhelpers.LocalhostCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid partner & cert names", func() {
						Partner = partner.Name
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then it should display the cert's info", func() {
								c := rest.FromCrypto(cert)
								So(getOutput(), ShouldEqual, certInfoString(c))
							})
						})
					})

					Convey("Given an invalid partner name", func() {
						Partner = "tutu"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "partner 'tutu' not found")
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						Partner = partner.Name
						args := []string{"tutu"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'tutu' not found")
							})
						})
					})
				})

				Convey("Given an account with a certificate", func() {
					account := &model.RemoteAccount{
						RemoteAgentID: partner.ID,
						Login:         "foo",
						Password:      "password",
					}
					So(db.Insert(account).Run(), ShouldBeNil)
					cert := &model.Crypto{
						RemoteAccountID: utils.NewNullInt64(account.ID),
						Name:            "account_cert",
						PrivateKey:      testhelpers.ClientFooKey,
						Certificate:     testhelpers.ClientFooCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid account, partner & cert names", func() {
						Partner = partner.Name
						RemoteAccount = account.Login
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then it should display the cert's info", func() {
								c := rest.FromCrypto(cert)
								So(getOutput(), ShouldEqual, certInfoString(c))
							})
						})
					})

					Convey("Given an invalid partner name", func() {
						Partner = "tutu"
						RemoteAccount = account.Login
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "partner 'tutu' not found")
							})
						})
					})

					Convey("Given an invalid account name", func() {
						Partner = partner.Name
						RemoteAccount = "tutu"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "no account 'tutu' found for partner "+partner.Name)
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						Partner = partner.Name
						RemoteAccount = account.Login
						args := []string{"tutu"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'tutu' not found")
							})
						})
					})
				})
			})

			Convey("Given a server", func() {
				server := &model.LocalAgent{
					Name:     "server",
					Protocol: testProto1,
					Address:  "localhost:6666",
				}
				So(db.Insert(server).Run(), ShouldBeNil)

				Convey("Given a server certificate", func() {
					cert := &model.Crypto{
						LocalAgentID: utils.NewNullInt64(server.ID),
						Name:         "server_cert",
						PrivateKey:   testhelpers.LocalhostKey,
						Certificate:  testhelpers.LocalhostCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid server & cert names", func() {
						Server = server.Name
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then it should display the cert's info", func() {
								c := rest.FromCrypto(cert)
								So(getOutput(), ShouldEqual, certInfoString(c))
							})
						})
					})

					Convey("Given an invalid server name", func() {
						Server = "tutu"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "server 'tutu' not found")
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						Server = server.Name
						args := []string{"tutu"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'tutu' not found")
							})
						})
					})
				})

				Convey("Given an account with a certificate", func() {
					account := &model.LocalAccount{
						LocalAgentID: server.ID,
						Login:        "foo",
						PasswordHash: hash("password"),
					}
					So(db.Insert(account).Run(), ShouldBeNil)
					cert := &model.Crypto{
						LocalAccountID: utils.NewNullInt64(account.ID),
						Name:           "account_cert",
						Certificate:    testhelpers.ClientFooCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid account, server & cert names", func() {
						Server = server.Name
						LocalAccount = account.Login
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then it should display the cert's info", func() {
								c := rest.FromCrypto(cert)
								So(getOutput(), ShouldEqual, certInfoString(c))
							})
						})
					})

					Convey("Given an invalid partner name", func() {
						Server = "tutu"
						LocalAccount = account.Login
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "server 'tutu' not found")
							})
						})
					})

					Convey("Given an invalid account name", func() {
						Server = server.Name
						LocalAccount = "tutu"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "no account 'tutu' found for server "+server.Name)
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						Server = server.Name
						LocalAccount = account.Login
						args := []string{"tutu"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'tutu' not found")
							})
						})
					})
				})
			})
		})
	})
}

//nolint:maintidx //FIXME factorize the function if possible to improve maintainability
func TestAddCertificate(t *testing.T) {
	Convey("Testing the cert 'add' command", t, func(c C) {
		resetVars()
		out = testFile()
		command := &CertAdd{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			cPk := writeFile(testhelpers.ClientFooKey)
			cCrt := writeFile(testhelpers.ClientFooCert)
			sPk := writeFile(testhelpers.LocalhostKey)
			sCrt := writeFile(testhelpers.LocalhostCert)

			Convey("Given a partner", func() {
				partner := &model.RemoteAgent{
					Name:     "partner",
					Protocol: testProto1,
					Address:  "localhost:6666",
				}
				So(db.Insert(partner).Run(), ShouldBeNil)

				Convey("When adding a new certificate", func() {
					Convey("Given valid partner & flags", func() {
						Partner = partner.Name
						args := []string{
							"-n", "partner_cert",
							"-c", sCrt.Name(),
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then is should display a message saying the cert was added", func() {
								So(getOutput(), ShouldEqual, "The certificate "+command.Name+
									" was successfully added.\n")
							})

							Convey("Then the new cert should have been added", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)

								So(certs, ShouldContain, &model.Crypto{
									ID:            1,
									RemoteAgentID: utils.NewNullInt64(partner.ID),
									Name:          "partner_cert",
									Certificate:   testhelpers.LocalhostCert,
								})
							})
						})
					})

					Convey("Given an invalid partner", func() {
						Partner = "tutu"
						args := []string{
							"-n", "partner_cert",
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then is should return an error", func() {
								So(err, ShouldBeError, "partner 'tutu' not found")
							})

							Convey("Then the new cert should NOT have been added", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldBeEmpty)
							})
						})
					})
				})

				Convey("Given a partner account", func() {
					account := &model.RemoteAccount{
						RemoteAgentID: partner.ID,
						Login:         "foo",
						Password:      "password",
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					Convey("When adding a new certificate", func() {
						Convey("Given valid account, partner & flags", func() {
							Partner = partner.Name
							RemoteAccount = account.Login
							args := []string{
								"-n", "account_cert",
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								So(command.Execute(params), ShouldBeNil)

								Convey("Then is should display a message saying the cert was added", func() {
									So(getOutput(), ShouldEqual, "The certificate "+command.Name+
										" was successfully added.\n")
								})

								Convey("Then the new cert should have been added", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)

									So(certs, ShouldContain, &model.Crypto{
										ID:              1,
										RemoteAccountID: utils.NewNullInt64(account.ID),
										Name:            "account_cert",
										PrivateKey:      testhelpers.ClientFooKey,
										Certificate:     testhelpers.ClientFooCert,
									})
								})
							})
						})

						Convey("Given an invalid partner", func() {
							Partner = "tutu"
							RemoteAccount = account.Login
							args := []string{
								"-n", "account_cert",
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "partner 'tutu' not found")
								})

								Convey("Then the new cert should NOT have been added", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldBeEmpty)
								})
							})
						})

						Convey("Given an invalid account", func() {
							Partner = partner.Name
							RemoteAccount = "tutu"
							args := []string{
								"-n", "account_cert",
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "no account 'tutu' found for partner "+partner.Name)
								})

								Convey("Then the new cert should NOT have been added", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldBeEmpty)
								})
							})
						})
					})
				})
			})

			Convey("Given a server", func() {
				server := &model.LocalAgent{
					Name:        "server",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(server).Run(), ShouldBeNil)

				Convey("When adding a new certificate", func() {
					Convey("Given valid server & flags", func() {
						Server = server.Name
						args := []string{
							"-n", "server_cert",
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then is should display a message saying the cert was added", func() {
								So(getOutput(), ShouldEqual, rest.ServerCertRestartRequiredMsg+
									"\nThe certificate "+command.Name+" was successfully added.\n")
							})

							Convey("Then the new cert should have been added", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)

								So(certs, ShouldContain, &model.Crypto{
									ID:           1,
									LocalAgentID: utils.NewNullInt64(server.ID),
									Name:         "server_cert",
									PrivateKey:   testhelpers.LocalhostKey,
									Certificate:  testhelpers.LocalhostCert,
								})
							})
						})
					})

					Convey("Given an invalid server", func() {
						Server = "tutu"
						args := []string{
							"-n", "server_cert",
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then is should return an error", func() {
								So(err, ShouldBeError, "server 'tutu' not found")
							})

							Convey("Then the new cert should NOT have been added", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldBeEmpty)
							})
						})
					})
				})

				Convey("Given a server account", func() {
					account := &model.LocalAccount{
						LocalAgentID: server.ID,
						Login:        "foo",
						PasswordHash: hash("password"),
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					Convey("When adding a new certificate", func() {
						Convey("Given valid account, server & flags", func() {
							Server = server.Name
							LocalAccount = account.Login
							args := []string{
								"-n", "account_cert",
								"-c", cCrt.Name(),
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								So(command.Execute(params), ShouldBeNil)

								Convey("Then is should display a message saying the cert was added", func() {
									So(getOutput(), ShouldEqual, "The certificate "+command.Name+
										" was successfully added.\n")
								})

								Convey("Then the new cert should have been added", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)

									So(certs, ShouldContain, &model.Crypto{
										ID:             1,
										LocalAccountID: utils.NewNullInt64(account.ID),
										Name:           "account_cert",
										Certificate:    testhelpers.ClientFooCert,
									})
								})
							})
						})

						Convey("Given an invalid server", func() {
							Server = "tutu"
							LocalAccount = account.Login
							args := []string{
								"-n", "account_cert",
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "server 'tutu' not found")
								})

								Convey("Then the new cert should NOT have been added", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldBeEmpty)
								})
							})
						})

						Convey("Given an invalid account", func() {
							Server = server.Name
							LocalAccount = "tutu"
							args := []string{
								"-n", "account_cert",
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "no account 'tutu' found for server "+server.Name)
								})

								Convey("Then the new cert should NOT have been added", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldBeEmpty)
								})
							})
						})
					})
				})
			})
		})
	})
}

//nolint:maintidx //FIXME factorize the function if possible to improve maintainability
func TestDeleteCertificate(t *testing.T) {
	Convey("Testing the certificate 'delete' command", t, func() {
		resetVars()
		out = testFile()
		command := &CertDelete{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a partner", func() {
				partner := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(partner).Run(), ShouldBeNil)

				Convey("Given a partner certificate", func() {
					cert := &model.Crypto{
						RemoteAgentID: utils.NewNullInt64(partner.ID),
						Name:          "partner_cert",
						Certificate:   testhelpers.LocalhostCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid partner & cert names", func() {
						Partner = partner.Name
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then it should say the cert was deleted", func() {
								So(getOutput(), ShouldEqual, "The certificate "+
									cert.Name+" was successfully deleted.\n")
							})

							Convey("Then the cert should have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldBeEmpty)
							})
						})
					})

					Convey("Given an invalid partner name", func() {
						Partner = "tutu"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "partner 'tutu' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, cert)
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						Partner = partner.Name
						args := []string{"tutu"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'tutu' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, cert)
							})
						})
					})
				})

				Convey("Given an account with a certificate", func() {
					account := &model.RemoteAccount{
						RemoteAgentID: partner.ID,
						Login:         "foo",
						Password:      "password",
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					cert := &model.Crypto{
						RemoteAccountID: utils.NewNullInt64(account.ID),
						Name:            "account_cert",
						PrivateKey:      testhelpers.ClientFooKey,
						Certificate:     testhelpers.ClientFooCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid account, partner & cert names", func() {
						Partner = partner.Name
						RemoteAccount = account.Login
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then it say the cert was deleted", func() {
								So(getOutput(), ShouldEqual, "The certificate "+
									cert.Name+" was successfully deleted.\n")
							})

							Convey("Then the cert should have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldBeEmpty)
							})
						})
					})

					Convey("Given an invalid partner name", func() {
						Partner = "tutu"
						RemoteAccount = account.Login
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "partner 'tutu' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, cert)
							})
						})
					})

					Convey("Given an invalid account name", func() {
						Partner = partner.Name
						RemoteAccount = "tutu"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "no account 'tutu' found for partner "+partner.Name)
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, cert)
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						Partner = partner.Name
						RemoteAccount = account.Login
						args := []string{"tutu"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'tutu' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, cert)
							})
						})
					})
				})
			})

			Convey("Given a server", func() {
				server := &model.LocalAgent{
					Name:        "server",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(server).Run(), ShouldBeNil)

				Convey("Given a server certificate", func() {
					cert := &model.Crypto{
						LocalAgentID: utils.NewNullInt64(server.ID),
						Name:         "server_cert",
						PrivateKey:   testhelpers.LocalhostKey,
						Certificate:  testhelpers.LocalhostCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid server & cert names", func() {
						Server = server.Name
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then it should say the cert was deleted", func() {
								So(getOutput(), ShouldEqual, "The certificate "+
									cert.Name+" was successfully deleted.\n")
							})

							Convey("Then the cert should have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldBeEmpty)
							})
						})
					})

					Convey("Given an invalid server name", func() {
						Server = "tutu"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "server 'tutu' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, cert)
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						Server = server.Name
						args := []string{"tutu"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'tutu' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, cert)
							})
						})
					})
				})

				Convey("Given an account with a certificate", func() {
					account := &model.LocalAccount{
						LocalAgentID: server.ID,
						Login:        "foo",
						PasswordHash: hash("password"),
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					cert := &model.Crypto{
						LocalAccountID: utils.NewNullInt64(account.ID),
						Name:           "account_cert",
						Certificate:    testhelpers.ClientFooCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid account, server & cert names", func() {
						Server = server.Name
						LocalAccount = account.Login
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then it say the cert was deleted", func() {
								So(getOutput(), ShouldEqual, "The certificate "+
									cert.Name+" was successfully deleted.\n")
							})

							Convey("Then the cert should have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldBeEmpty)
							})
						})
					})

					Convey("Given an invalid server name", func() {
						Server = "tutu"
						LocalAccount = account.Login
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "server 'tutu' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, cert)
							})
						})
					})

					Convey("Given an invalid account name", func() {
						Server = server.Name
						LocalAccount = "tutu"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "no account 'tutu' found for server "+server.Name)
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, cert)
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						Server = server.Name
						LocalAccount = account.Login
						args := []string{"tutu"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'tutu' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, cert)
							})
						})
					})
				})
			})
		})
	})
}

//nolint:maintidx //FIXME factorize the function if possible to improve maintainability
func TestListCertificate(t *testing.T) {
	Convey("Testing the certificate 'list' command", t, func() {
		resetVars()
		out = testFile()
		command := &CertList{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a partner", func() {
				partner := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(partner).Run(), ShouldBeNil)

				Convey("Given a partner certificate", func() {
					cert1 := &model.Crypto{
						RemoteAgentID: utils.NewNullInt64(partner.ID),
						Name:          "partner_cert_1",
						Certificate:   testhelpers.LocalhostCert,
					}
					So(db.Insert(cert1).Run(), ShouldBeNil)
					cert2 := &model.Crypto{
						RemoteAgentID: utils.NewNullInt64(partner.ID),
						Name:          "partner_cert_2",
						Certificate:   testhelpers.OtherLocalhostCert,
					}
					So(db.Insert(cert2).Run(), ShouldBeNil)

					c1 := rest.FromCrypto(cert1)
					c2 := rest.FromCrypto(cert2)

					Convey("Given no flags", func() {
						Partner = partner.Name
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then it should display the certificates", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c1)+certInfoString(c2))
							})
						})
					})

					Convey("Given an invalid partner name", func() {
						Partner = "tutu"
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "partner 'tutu' not found")
							})
						})
					})

					Convey("Given a 'limit' parameter of 1", func() {
						Partner = partner.Name
						args := []string{"-l", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							command.Execute(params)

							Convey("Then it should only display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c1))
							})
						})
					})

					Convey("Given a 'offset' parameter of 1", func() {
						Partner = partner.Name
						args := []string{"-o", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							command.Execute(params)

							Convey("Then it should NOT display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c2))
							})
						})
					})

					Convey("Given a 'sort' parameter of 'name-'", func() {
						Partner = partner.Name
						args := []string{"-s", "name-"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							command.Execute(params)

							Convey("Then it should display the certificates in reverse", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c2)+certInfoString(c1))
							})
						})
					})
				})

				Convey("Given an account with a certificate", func() {
					account := &model.RemoteAccount{
						RemoteAgentID: partner.ID,
						Login:         "foo",
						Password:      "password",
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					cert1 := &model.Crypto{
						RemoteAccountID: utils.NewNullInt64(account.ID),
						Name:            "account_cert_1",
						PrivateKey:      testhelpers.ClientFooKey,
						Certificate:     testhelpers.ClientFooCert,
					}
					So(db.Insert(cert1).Run(), ShouldBeNil)

					cert2 := &model.Crypto{
						RemoteAccountID: utils.NewNullInt64(account.ID),
						Name:            "account_cert_2",
						PrivateKey:      testhelpers.ClientFooKey2,
						Certificate:     testhelpers.ClientFooCert2,
					}
					So(db.Insert(cert2).Run(), ShouldBeNil)

					c1 := rest.FromCrypto(cert1)
					c2 := rest.FromCrypto(cert2)

					Convey("Given no flags", func() {
						Partner = partner.Name
						RemoteAccount = account.Login
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then it should display the certificates", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c1)+certInfoString(c2))
							})
						})
					})

					Convey("Given an invalid partner name", func() {
						Partner = "tutu"
						RemoteAccount = account.Login
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "partner 'tutu' not found")
							})
						})
					})

					Convey("Given an invalid account name", func() {
						Partner = partner.Name
						RemoteAccount = "tutu"
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "no account 'tutu' found for partner "+partner.Name)
							})
						})
					})

					Convey("Given a 'limit' parameter of 1", func() {
						Partner = partner.Name
						RemoteAccount = account.Login
						args := []string{"-l", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							command.Execute(params)

							Convey("Then it should only display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c1))
							})
						})
					})

					Convey("Given a 'offset' parameter of 1", func() {
						Partner = partner.Name
						RemoteAccount = account.Login
						args := []string{"-o", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							command.Execute(params)

							Convey("Then it should NOT display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c2))
							})
						})
					})

					Convey("Given a 'sort' parameter of 'name-'", func() {
						Partner = partner.Name
						RemoteAccount = account.Login
						args := []string{"-s", "name-"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							command.Execute(params)

							Convey("Then it should display the certificates in reverse", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c2)+certInfoString(c1))
							})
						})
					})
				})
			})

			Convey("Given a server", func() {
				server := &model.LocalAgent{
					Name:        "server",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(server).Run(), ShouldBeNil)

				Convey("Given a server certificate", func() {
					cert1 := &model.Crypto{
						LocalAgentID: utils.NewNullInt64(server.ID),
						Name:         "server_cert_1",
						PrivateKey:   testhelpers.LocalhostKey,
						Certificate:  testhelpers.LocalhostCert,
					}
					So(db.Insert(cert1).Run(), ShouldBeNil)

					cert2 := &model.Crypto{
						LocalAgentID: utils.NewNullInt64(server.ID),
						Name:         "server_cert_2",
						PrivateKey:   testhelpers.OtherLocalhostKey,
						Certificate:  testhelpers.OtherLocalhostCert,
					}
					So(db.Insert(cert2).Run(), ShouldBeNil)

					c1 := rest.FromCrypto(cert1)
					c2 := rest.FromCrypto(cert2)

					Convey("Given no flags", func() {
						Server = server.Name
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then it should display the certificates", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c1)+certInfoString(c2))
							})
						})
					})

					Convey("Given an invalid server name", func() {
						Server = "tutu"
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "server 'tutu' not found")
							})
						})
					})

					Convey("Given a 'limit' parameter of 1", func() {
						Server = server.Name
						args := []string{"-l", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							command.Execute(params)

							Convey("Then it should only display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c1))
							})
						})
					})

					Convey("Given a 'offset' parameter of 1", func() {
						Server = server.Name
						args := []string{"-o", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							command.Execute(params)

							Convey("Then it should NOT display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c2))
							})
						})
					})

					Convey("Given a 'sort' parameter of 'name-'", func() {
						Server = server.Name
						args := []string{"-s", "name-"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							command.Execute(params)

							Convey("Then it should display the certificates in reverse", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c2)+certInfoString(c1))
							})
						})
					})
				})

				Convey("Given an account with a certificate", func() {
					account := &model.LocalAccount{
						LocalAgentID: server.ID,
						Login:        "foo",
						PasswordHash: hash("password"),
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					cert1 := &model.Crypto{
						LocalAccountID: utils.NewNullInt64(account.ID),
						Name:           "account_cert_1",
						Certificate:    testhelpers.ClientFooCert,
					}
					So(db.Insert(cert1).Run(), ShouldBeNil)

					cert2 := &model.Crypto{
						LocalAccountID: utils.NewNullInt64(account.ID),
						Name:           "account_cert_2",
						Certificate:    testhelpers.ClientFooCert2,
					}
					So(db.Insert(cert2).Run(), ShouldBeNil)

					c1 := rest.FromCrypto(cert1)
					c2 := rest.FromCrypto(cert2)

					Convey("Given no flags", func() {
						Server = server.Name
						LocalAccount = account.Login
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then it should display the certificates", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c1)+certInfoString(c2))
							})
						})
					})

					Convey("Given an invalid server name", func() {
						Server = "tutu"
						LocalAccount = account.Login
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "server 'tutu' not found")
							})
						})
					})

					Convey("Given an invalid account name", func() {
						Server = server.Name
						LocalAccount = "tutu"
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "no account 'tutu' found for server "+server.Name)
							})
						})
					})

					Convey("Given a 'limit' parameter of 1", func() {
						Server = server.Name
						LocalAccount = account.Login
						args := []string{"-l", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							command.Execute(params)

							Convey("Then it should only display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c1))
							})
						})
					})

					Convey("Given a 'offset' parameter of 1", func() {
						Server = server.Name
						LocalAccount = account.Login
						args := []string{"-o", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							command.Execute(params)

							Convey("Then it should NOT display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c2))
							})
						})
					})

					Convey("Given a 'sort' parameter of 'name-'", func() {
						Server = server.Name
						LocalAccount = account.Login
						args := []string{"-s", "name-"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							command.Execute(params)

							Convey("Then it should display the certificates in reverse", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c2)+certInfoString(c1))
							})
						})
					})
				})
			})
		})
	})
}

//nolint:maintidx //FIXME factorize the function if possible to improve maintainability
func TestUpdateCertificate(t *testing.T) {
	Convey("Testing the certificate 'update' command", t, func() {
		resetVars()
		out = testFile()
		command := &CertUpdate{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			cPk := writeFile(testhelpers.ClientFooKey)
			cCrt := writeFile(testhelpers.ClientFooCert)
			sPk := writeFile(testhelpers.LocalhostKey)
			sCrt := writeFile(testhelpers.LocalhostCert)

			Convey("Given a partner", func() {
				partner := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(partner).Run(), ShouldBeNil)

				Convey("When updating the certificate", func() {
					originalCert := &model.Crypto{
						RemoteAgentID: utils.NewNullInt64(partner.ID),
						Name:          "partner_cert",
						Certificate:   testhelpers.LocalhostCert,
					}
					So(db.Insert(originalCert).Run(), ShouldBeNil)

					Convey("Given valid partner, certificate & flags", func() {
						Partner = partner.Name
						args := []string{
							"-c", sCrt.Name(),
							originalCert.Name,
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then is should display a message saying the "+
								"cert was added", func() {
								So(getOutput(), ShouldEqual, "The certificate "+
									originalCert.Name+" was successfully updated.\n")
							})

							Convey("Then the cert should have been updated", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)

								So(certs, ShouldContain, &model.Crypto{
									ID:            originalCert.ID,
									RemoteAgentID: utils.NewNullInt64(partner.ID),
									Name:          "partner_cert",
									Certificate:   testhelpers.LocalhostCert,
								})
							})
						})
					})

					Convey("Given an invalid partner name", func() {
						Partner = "tutu"
						args := []string{
							"-n", "partner_cert",
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
							originalCert.Name,
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then is should return an error", func() {
								So(err, ShouldBeError, "partner 'tutu' not found")
							})

							Convey("Then the new cert should NOT have been changed", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, originalCert)
							})
						})
					})

					Convey("Given an invalid certificate name", func() {
						Partner = partner.Name
						args := []string{
							"-n", "partner_cert",
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
							"tutu",
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then is should return an error", func() {
								So(err, ShouldBeError, "certificate 'tutu' not found")
							})

							Convey("Then the new cert should NOT have been changed", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, originalCert)
							})
						})
					})
				})

				Convey("Given a partner account", func() {
					account := &model.RemoteAccount{
						RemoteAgentID: partner.ID,
						Login:         "foo",
						Password:      "password",
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					Convey("When updating the certificate", func() {
						originalCert := &model.Crypto{
							RemoteAccountID: utils.NewNullInt64(account.ID),
							Name:            "account_cert",
							PrivateKey:      testhelpers.ClientFooKey,
							Certificate:     testhelpers.ClientFooCert,
						}
						So(db.Insert(originalCert).Run(), ShouldBeNil)

						Convey("Given valid account, partner & flags", func() {
							Partner = partner.Name
							RemoteAccount = account.Login
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								originalCert.Name,
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								So(command.Execute(params), ShouldBeNil)

								Convey("Then is should display a message saying "+
									"the cert was added", func() {
									So(getOutput(), ShouldEqual, "The certificate "+
										originalCert.Name+" was successfully updated.\n")
								})

								Convey("Then the cert should have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)

									So(certs, ShouldContain, &model.Crypto{
										ID:              originalCert.ID,
										RemoteAccountID: utils.NewNullInt64(account.ID),
										Name:            "account_cert",
										PrivateKey:      testhelpers.ClientFooKey,
										Certificate:     testhelpers.ClientFooCert,
									})
								})
							})
						})

						Convey("Given an invalid partner name", func() {
							Partner = "tutu"
							RemoteAccount = account.Login
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								originalCert.Name,
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "partner 'tutu' not found")
								})

								Convey("Then the cert should NOT have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, originalCert)
								})
							})
						})

						Convey("Given an invalid account name", func() {
							Partner = partner.Name
							RemoteAccount = "tutu"
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								originalCert.Name,
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "no account 'tutu' "+
										"found for partner "+partner.Name)
								})

								Convey("Then the cert should NOT have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, originalCert)
								})
							})
						})

						Convey("Given an invalid certificate name", func() {
							Partner = partner.Name
							RemoteAccount = account.Login
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								"tutu",
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "certificate 'tutu' not found")
								})

								Convey("Then the cert should NOT have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, originalCert)
								})
							})
						})
					})
				})
			})

			Convey("Given a server", func() {
				server := &model.LocalAgent{
					Name:        "server",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(server).Run(), ShouldBeNil)

				Convey("When updating the certificate", func() {
					originalCert := &model.Crypto{
						LocalAgentID: utils.NewNullInt64(server.ID),
						Name:         "server_cert",
						PrivateKey:   testhelpers.LocalhostKey,
						Certificate:  testhelpers.LocalhostCert,
					}
					So(db.Insert(originalCert).Run(), ShouldBeNil)

					Convey("Given valid server, certificate & flags", func() {
						Server = server.Name
						args := []string{
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
							originalCert.Name,
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then is should display a message saying "+
								"the cert was added", func() {
								So(getOutput(), ShouldEqual, rest.ServerCertRestartRequiredMsg+
									"\nThe certificate "+originalCert.Name+" was successfully updated.\n")
							})

							Convey("Then the cert should have been updated", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)

								So(certs, ShouldContain, &model.Crypto{
									ID:           originalCert.ID,
									LocalAgentID: utils.NewNullInt64(server.ID),
									Name:         "server_cert",
									PrivateKey:   testhelpers.LocalhostKey,
									Certificate:  testhelpers.LocalhostCert,
								})
							})
						})
					})

					Convey("Given an invalid server name", func() {
						Server = "tutu"
						args := []string{
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
							originalCert.Name,
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then is should return an error", func() {
								So(err, ShouldBeError, "server 'tutu' not found")
							})

							Convey("Then the new cert should NOT have been changed", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, originalCert)
							})
						})
					})

					Convey("Given an invalid certificate name", func() {
						Server = server.Name
						args := []string{
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
							"tutu",
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then is should return an error", func() {
								So(err, ShouldBeError, "certificate 'tutu' not found")
							})

							Convey("Then the new cert should NOT have been changed", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, originalCert)
							})
						})
					})
				})

				Convey("Given a server account", func() {
					account := &model.LocalAccount{
						LocalAgentID: server.ID,
						Login:        "foo",
						PasswordHash: hash("password"),
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					Convey("When updating the certificate", func() {
						originalCert := &model.Crypto{
							LocalAccountID: utils.NewNullInt64(account.ID),
							Name:           "account_cert",
							Certificate:    testhelpers.ClientFooCert,
						}
						So(db.Insert(originalCert).Run(), ShouldBeNil)

						Convey("Given valid account, server & flags", func() {
							Server = server.Name
							LocalAccount = account.Login
							args := []string{
								"-c", cCrt.Name(),
								originalCert.Name,
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								So(command.Execute(params), ShouldBeNil)

								Convey("Then is should display a message saying "+
									"the cert was added", func() {
									So(getOutput(), ShouldEqual, "The certificate "+
										originalCert.Name+" was successfully updated.\n")
								})

								Convey("Then the cert should have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)

									So(certs, ShouldContain, &model.Crypto{
										ID:             originalCert.ID,
										LocalAccountID: utils.NewNullInt64(account.ID),
										Name:           "account_cert",
										Certificate:    testhelpers.ClientFooCert,
									})
								})
							})
						})

						Convey("Given an invalid server name", func() {
							Server = "tutu"
							LocalAccount = account.Login
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								originalCert.Name,
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "server 'tutu' not found")
								})

								Convey("Then the cert should NOT have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, originalCert)
								})
							})
						})

						Convey("Given an invalid account name", func() {
							Server = server.Name
							LocalAccount = "tutu"
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								originalCert.Name,
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "no account 'tutu' "+
										"found for server "+server.Name)
								})

								Convey("Then the cert should NOT have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, originalCert)
								})
							})
						})

						Convey("Given an invalid certificate name", func() {
							Server = server.Name
							LocalAccount = account.Login
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								"tutu",
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "certificate 'tutu' not found")
								})

								Convey("Then the cert should NOT have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, originalCert)
								})
							})
						})
					})
				})
			})
		})
	})
}
