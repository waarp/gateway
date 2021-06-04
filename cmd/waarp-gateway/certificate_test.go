package main

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func certInfoString(c *api.OutCrypto) string {
	return "● Certificate " + c.Name + "\n" +
		"    Private key: " + c.PrivateKey + "\n" +
		"    Public key:  " + c.PublicKey + "\n" +
		"    Content:     " + c.Certificate + "\n"
}

func TestGetCertificate(t *testing.T) {

	Convey("Testing the certificate 'get' command", t, func() {
		out = testFile()
		command := &certGet{}
		commandLine = options{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a partner", func() {
				partner := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(partner).Run(), ShouldBeNil)

				Convey("Given a partner certificate", func() {
					cert := &model.Crypto{
						OwnerType:   partner.TableName(),
						OwnerID:     partner.ID,
						Name:        "partner_cert",
						Certificate: testhelpers.LocalhostCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid partner & cert names", func() {
						commandLine.Partner.Cert.Args.Partner = partner.Name
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
						commandLine.Partner.Cert.Args.Partner = "toto"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "partner 'toto' not found")
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						commandLine.Partner.Cert.Args.Partner = partner.Name
						args := []string{"toto"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'toto' not found")
							})
						})
					})
				})

				Convey("Given an account with a certificate", func() {
					account := &model.RemoteAccount{
						RemoteAgentID: partner.ID,
						Login:         "rem_account",
						Password:      "password",
					}
					So(db.Insert(account).Run(), ShouldBeNil)
					cert := &model.Crypto{
						OwnerType:   account.TableName(),
						OwnerID:     account.ID,
						Name:        "account_cert",
						PrivateKey:  testhelpers.ClientKey,
						Certificate: testhelpers.ClientCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid account, partner & cert names", func() {
						commandLine.Account.Remote.Args.Partner = partner.Name
						commandLine.Account.Remote.Cert.Args.Account = account.Login
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
						commandLine.Account.Remote.Args.Partner = "toto"
						commandLine.Account.Remote.Cert.Args.Account = account.Login
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "partner 'toto' not found")
							})
						})
					})

					Convey("Given an invalid account name", func() {
						commandLine.Account.Remote.Args.Partner = partner.Name
						commandLine.Account.Remote.Cert.Args.Account = "toto"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "no account 'toto' found for partner "+partner.Name)
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						commandLine.Account.Remote.Args.Partner = partner.Name
						commandLine.Account.Remote.Cert.Args.Account = account.Login
						args := []string{"toto"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'toto' not found")
							})
						})
					})
				})
			})

			Convey("Given a server", func() {
				server := &model.LocalAgent{
					Name:        "server",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(server).Run(), ShouldBeNil)

				Convey("Given a server certificate", func() {
					cert := &model.Crypto{
						OwnerType:   server.TableName(),
						OwnerID:     server.ID,
						Name:        "server_cert",
						PrivateKey:  testhelpers.LocalhostKey,
						Certificate: testhelpers.LocalhostCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid server & cert names", func() {
						commandLine.Server.Cert.Args.Server = server.Name
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
						commandLine.Server.Cert.Args.Server = "toto"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "server 'toto' not found")
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						commandLine.Server.Cert.Args.Server = server.Name
						args := []string{"toto"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'toto' not found")
							})
						})
					})
				})

				Convey("Given an account with a certificate", func() {
					account := &model.LocalAccount{
						LocalAgentID: server.ID,
						Login:        "loc_account",
						PasswordHash: hash("password"),
					}
					So(db.Insert(account).Run(), ShouldBeNil)
					cert := &model.Crypto{
						OwnerType:   account.TableName(),
						OwnerID:     account.ID,
						Name:        "account_cert",
						Certificate: testhelpers.ClientCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid account, server & cert names", func() {
						commandLine.Account.Local.Args.Server = server.Name
						commandLine.Account.Local.Cert.Args.Account = account.Login
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
						commandLine.Account.Local.Args.Server = "toto"
						commandLine.Account.Local.Cert.Args.Account = account.Login
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "server 'toto' not found")
							})
						})
					})

					Convey("Given an invalid account name", func() {
						commandLine.Account.Local.Args.Server = server.Name
						commandLine.Account.Local.Cert.Args.Account = "toto"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "no account 'toto' found for server "+server.Name)
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						commandLine.Account.Local.Args.Server = server.Name
						commandLine.Account.Local.Cert.Args.Account = account.Login
						args := []string{"toto"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'toto' not found")
							})
						})
					})
				})
			})
		})
	})
}

func TestAddCertificate(t *testing.T) {

	Convey("Testing the cert 'add' command", t, func(c C) {
		out = testFile()
		command := &certAdd{}
		commandLine = options{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			cPk := writeFile(testhelpers.ClientKey)
			cCrt := writeFile(testhelpers.ClientCert)
			sPk := writeFile(testhelpers.LocalhostKey)
			sCrt := writeFile(testhelpers.LocalhostCert)

			Convey("Given a partner", func() {
				partner := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(partner).Run(), ShouldBeNil)

				Convey("When adding a new certificate", func() {

					Convey("Given valid partner & flags", func() {
						commandLine.Partner.Cert.Args.Partner = partner.Name
						args := []string{"-n", "partner_cert",
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

								exp := model.Crypto{
									ID:          1,
									OwnerType:   partner.TableName(),
									OwnerID:     partner.ID,
									Name:        "partner_cert",
									Certificate: testhelpers.LocalhostCert,
								}
								So(certs, ShouldContain, exp)
							})
						})
					})

					Convey("Given an invalid partner", func() {
						commandLine.Partner.Cert.Args.Partner = "toto"
						args := []string{"-n", "partner_cert",
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then is should return an error", func() {
								So(err, ShouldBeError, "partner 'toto' not found")
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
						Login:         "rem_account",
						Password:      "password",
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					Convey("When adding a new certificate", func() {

						Convey("Given valid account, partner & flags", func() {
							commandLine.Account.Remote.Args.Partner = partner.Name
							commandLine.Account.Remote.Cert.Args.Account = account.Login
							args := []string{"-n", "account_cert",
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

									exp := model.Crypto{
										ID:          1,
										OwnerType:   account.TableName(),
										OwnerID:     account.ID,
										Name:        "account_cert",
										PrivateKey:  testhelpers.ClientKey,
										Certificate: testhelpers.ClientCert,
									}
									So(certs, ShouldContain, exp)
								})
							})
						})

						Convey("Given an invalid partner", func() {
							commandLine.Account.Remote.Args.Partner = "toto"
							commandLine.Account.Remote.Cert.Args.Account = account.Login
							args := []string{"-n", "account_cert",
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "partner 'toto' not found")
								})

								Convey("Then the new cert should NOT have been added", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldBeEmpty)
								})
							})
						})

						Convey("Given an invalid account", func() {
							commandLine.Account.Remote.Args.Partner = partner.Name
							commandLine.Account.Remote.Cert.Args.Account = "toto"
							args := []string{"-n", "account_cert",
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "no account 'toto' found for partner "+partner.Name)
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
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(server).Run(), ShouldBeNil)

				Convey("When adding a new certificate", func() {

					Convey("Given valid server & flags", func() {
						commandLine.Server.Cert.Args.Server = server.Name
						args := []string{"-n", "server_cert",
							"-p", sPk.Name(),
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
								So(certs, ShouldNotBeEmpty)

								exp := model.Crypto{
									ID:          1,
									OwnerType:   server.TableName(),
									OwnerID:     server.ID,
									Name:        "server_cert",
									PrivateKey:  testhelpers.LocalhostKey,
									Certificate: testhelpers.LocalhostCert,
								}
								So(certs, ShouldContain, exp)
							})
						})
					})

					Convey("Given an invalid server", func() {
						commandLine.Server.Cert.Args.Server = "toto"
						args := []string{"-n", "server_cert",
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then is should return an error", func() {
								So(err, ShouldBeError, "server 'toto' not found")
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
						Login:        "loc_account",
						PasswordHash: hash("password"),
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					Convey("When adding a new certificate", func() {

						Convey("Given valid account, server & flags", func() {
							commandLine.Account.Local.Args.Server = server.Name
							commandLine.Account.Local.Cert.Args.Account = account.Login
							args := []string{"-n", "account_cert",
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

									exp := model.Crypto{
										ID:          1,
										OwnerType:   account.TableName(),
										OwnerID:     account.ID,
										Name:        "account_cert",
										Certificate: testhelpers.ClientCert,
									}
									So(certs, ShouldContain, exp)
								})
							})
						})

						Convey("Given an invalid server", func() {
							commandLine.Account.Local.Args.Server = "toto"
							commandLine.Account.Local.Cert.Args.Account = account.Login
							args := []string{"-n", "account_cert",
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "server 'toto' not found")
								})

								Convey("Then the new cert should NOT have been added", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldBeEmpty)
								})
							})
						})

						Convey("Given an invalid account", func() {
							commandLine.Account.Local.Args.Server = server.Name
							commandLine.Account.Local.Cert.Args.Account = "toto"
							args := []string{"-n", "account_cert",
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "no account 'toto' found for server "+server.Name)
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

func TestDeleteCertificate(t *testing.T) {

	Convey("Testing the certificate 'delete' command", t, func() {
		out = testFile()
		command := &certDelete{}
		commandLine = options{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a partner", func() {
				partner := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(partner).Run(), ShouldBeNil)

				Convey("Given a partner certificate", func() {
					cert := &model.Crypto{
						OwnerType:   partner.TableName(),
						OwnerID:     partner.ID,
						Name:        "partner_cert",
						Certificate: testhelpers.LocalhostCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid partner & cert names", func() {
						commandLine.Partner.Cert.Args.Partner = partner.Name
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
						commandLine.Partner.Cert.Args.Partner = "toto"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "partner 'toto' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						commandLine.Partner.Cert.Args.Partner = partner.Name
						args := []string{"toto"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'toto' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})
				})

				Convey("Given an account with a certificate", func() {
					account := &model.RemoteAccount{
						RemoteAgentID: partner.ID,
						Login:         "rem_account",
						Password:      "password",
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					cert := &model.Crypto{
						OwnerType:   account.TableName(),
						OwnerID:     account.ID,
						Name:        "account_cert",
						PrivateKey:  testhelpers.ClientKey,
						Certificate: testhelpers.ClientCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid account, partner & cert names", func() {
						commandLine.Account.Remote.Args.Partner = partner.Name
						commandLine.Account.Remote.Cert.Args.Account = account.Login
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
						commandLine.Account.Remote.Args.Partner = "toto"
						commandLine.Account.Remote.Cert.Args.Account = account.Login
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "partner 'toto' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})

					Convey("Given an invalid account name", func() {
						commandLine.Account.Remote.Args.Partner = partner.Name
						commandLine.Account.Remote.Cert.Args.Account = "toto"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "no account 'toto' found for partner "+partner.Name)
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						commandLine.Account.Remote.Args.Partner = partner.Name
						commandLine.Account.Remote.Cert.Args.Account = account.Login
						args := []string{"toto"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'toto' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})
				})
			})

			Convey("Given a server", func() {
				server := &model.LocalAgent{
					Name:        "server",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(server).Run(), ShouldBeNil)

				Convey("Given a server certificate", func() {
					cert := &model.Crypto{
						OwnerType:   server.TableName(),
						OwnerID:     server.ID,
						Name:        "server_cert",
						PrivateKey:  testhelpers.LocalhostKey,
						Certificate: testhelpers.LocalhostCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid server & cert names", func() {
						commandLine.Server.Cert.Args.Server = server.Name
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
						commandLine.Server.Cert.Args.Server = "toto"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "server 'toto' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						commandLine.Server.Cert.Args.Server = server.Name
						args := []string{"toto"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'toto' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})
				})

				Convey("Given an account with a certificate", func() {
					account := &model.LocalAccount{
						LocalAgentID: server.ID,
						Login:        "loc_account",
						PasswordHash: hash("password"),
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					cert := &model.Crypto{
						OwnerType:   account.TableName(),
						OwnerID:     account.ID,
						Name:        "account_cert",
						Certificate: testhelpers.ClientCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid account, server & cert names", func() {
						commandLine.Account.Local.Args.Server = server.Name
						commandLine.Account.Local.Cert.Args.Account = account.Login
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
						commandLine.Account.Local.Args.Server = "toto"
						commandLine.Account.Local.Cert.Args.Account = account.Login
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "server 'toto' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})

					Convey("Given an invalid account name", func() {
						commandLine.Account.Local.Args.Server = server.Name
						commandLine.Account.Local.Cert.Args.Account = "toto"
						args := []string{cert.Name}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "no account 'toto' found for server "+server.Name)
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})

					Convey("Given an invalid cert name", func() {
						commandLine.Account.Local.Args.Server = server.Name
						commandLine.Account.Local.Cert.Args.Account = account.Login
						args := []string{"toto"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "certificate 'toto' not found")
							})

							Convey("Then the cert should NOT have been deleted", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})
				})
			})
		})
	})
}

func TestListCertificate(t *testing.T) {

	Convey("Testing the certificate 'list' command", t, func() {
		out = testFile()
		command := &certList{}
		commandLine = options{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a partner", func() {
				partner := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(partner).Run(), ShouldBeNil)

				Convey("Given a partner certificate", func() {
					cert1 := &model.Crypto{
						OwnerType:   partner.TableName(),
						OwnerID:     partner.ID,
						Name:        "partner_cert_1",
						Certificate: testhelpers.LocalhostCert,
					}
					So(db.Insert(cert1).Run(), ShouldBeNil)
					cert2 := &model.Crypto{
						OwnerType:   partner.TableName(),
						OwnerID:     partner.ID,
						Name:        "partner_cert_2",
						Certificate: testhelpers.LocalhostCert,
					}
					So(db.Insert(cert2).Run(), ShouldBeNil)

					c1 := rest.FromCrypto(cert1)
					c2 := rest.FromCrypto(cert2)

					Convey("Given no flags", func() {
						commandLine.Partner.Cert.Args.Partner = partner.Name
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
						commandLine.Partner.Cert.Args.Partner = "toto"
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "partner 'toto' not found")
							})
						})
					})

					Convey("Given a 'limit' parameter of 1", func() {
						commandLine.Partner.Cert.Args.Partner = partner.Name
						args := []string{"-l", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should only display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c1))
							})
						})
					})

					Convey("Given a 'offset' parameter of 1", func() {
						commandLine.Partner.Cert.Args.Partner = partner.Name
						args := []string{"-o", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should NOT display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c2))
							})
						})
					})

					Convey("Given a 'sort' parameter of 'name-'", func() {
						commandLine.Partner.Cert.Args.Partner = partner.Name
						args := []string{"-s", "name-"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

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
						Login:         "rem_account",
						Password:      "password",
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					cert1 := &model.Crypto{
						OwnerType:   account.TableName(),
						OwnerID:     account.ID,
						Name:        "account_cert_1",
						PrivateKey:  testhelpers.ClientKey,
						Certificate: testhelpers.ClientCert,
					}
					So(db.Insert(cert1).Run(), ShouldBeNil)

					cert2 := &model.Crypto{
						OwnerType:   account.TableName(),
						OwnerID:     account.ID,
						Name:        "account_cert_2",
						PrivateKey:  testhelpers.ClientKey,
						Certificate: testhelpers.ClientCert,
					}
					So(db.Insert(cert2).Run(), ShouldBeNil)

					c1 := rest.FromCrypto(cert1)
					c2 := rest.FromCrypto(cert2)

					Convey("Given no flags", func() {
						commandLine.Account.Remote.Args.Partner = partner.Name
						commandLine.Account.Remote.Cert.Args.Account = account.Login
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
						commandLine.Account.Remote.Args.Partner = "toto"
						commandLine.Account.Remote.Cert.Args.Account = account.Login
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "partner 'toto' not found")
							})
						})
					})

					Convey("Given an invalid account name", func() {
						commandLine.Account.Remote.Args.Partner = partner.Name
						commandLine.Account.Remote.Cert.Args.Account = "toto"
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "no account 'toto' found for partner "+partner.Name)
							})
						})
					})

					Convey("Given a 'limit' parameter of 1", func() {
						commandLine.Account.Remote.Args.Partner = partner.Name
						commandLine.Account.Remote.Cert.Args.Account = account.Login
						args := []string{"-l", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should only display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c1))
							})
						})
					})

					Convey("Given a 'offset' parameter of 1", func() {
						commandLine.Account.Remote.Args.Partner = partner.Name
						commandLine.Account.Remote.Cert.Args.Account = account.Login
						args := []string{"-o", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should NOT display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c2))
							})
						})
					})

					Convey("Given a 'sort' parameter of 'name-'", func() {
						commandLine.Account.Remote.Args.Partner = partner.Name
						commandLine.Account.Remote.Cert.Args.Account = account.Login
						args := []string{"-s", "name-"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

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
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(server).Run(), ShouldBeNil)

				Convey("Given a server certificate", func() {
					cert1 := &model.Crypto{
						OwnerType:   server.TableName(),
						OwnerID:     server.ID,
						Name:        "server_cert_1",
						PrivateKey:  testhelpers.LocalhostKey,
						Certificate: testhelpers.LocalhostCert,
					}
					So(db.Insert(cert1).Run(), ShouldBeNil)

					cert2 := &model.Crypto{
						OwnerType:   server.TableName(),
						OwnerID:     server.ID,
						Name:        "server_cert_2",
						PrivateKey:  testhelpers.LocalhostKey,
						Certificate: testhelpers.LocalhostCert,
					}
					So(db.Insert(cert2).Run(), ShouldBeNil)

					c1 := rest.FromCrypto(cert1)
					c2 := rest.FromCrypto(cert2)

					Convey("Given no flags", func() {
						commandLine.Server.Cert.Args.Server = server.Name
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
						commandLine.Server.Cert.Args.Server = "toto"
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "server 'toto' not found")
							})
						})
					})

					Convey("Given a 'limit' parameter of 1", func() {
						commandLine.Server.Cert.Args.Server = server.Name
						args := []string{"-l", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should only display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c1))
							})
						})
					})

					Convey("Given a 'offset' parameter of 1", func() {
						commandLine.Server.Cert.Args.Server = server.Name
						args := []string{"-o", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should NOT display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c2))
							})
						})
					})

					Convey("Given a 'sort' parameter of 'name-'", func() {
						commandLine.Server.Cert.Args.Server = server.Name
						args := []string{"-s", "name-"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

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
						Login:        "loc_account",
						PasswordHash: hash("password"),
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					cert1 := &model.Crypto{
						OwnerType:   account.TableName(),
						OwnerID:     account.ID,
						Name:        "account_cert_1",
						Certificate: testhelpers.ClientCert,
					}
					So(db.Insert(cert1).Run(), ShouldBeNil)

					cert2 := &model.Crypto{
						OwnerType:   account.TableName(),
						OwnerID:     account.ID,
						Name:        "account_cert_2",
						Certificate: testhelpers.ClientCert,
					}
					So(db.Insert(cert2).Run(), ShouldBeNil)

					c1 := rest.FromCrypto(cert1)
					c2 := rest.FromCrypto(cert2)

					Convey("Given no flags", func() {
						commandLine.Account.Local.Args.Server = server.Name
						commandLine.Account.Local.Cert.Args.Account = account.Login
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
						commandLine.Account.Local.Args.Server = "toto"
						commandLine.Account.Local.Cert.Args.Account = account.Login
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "server 'toto' not found")
							})
						})
					})

					Convey("Given an invalid account name", func() {
						commandLine.Account.Local.Args.Server = server.Name
						commandLine.Account.Local.Cert.Args.Account = "toto"
						args := []string{}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError, "no account 'toto' found for server "+server.Name)
							})
						})
					})

					Convey("Given a 'limit' parameter of 1", func() {
						commandLine.Account.Local.Args.Server = server.Name
						commandLine.Account.Local.Cert.Args.Account = account.Login
						args := []string{"-l", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should only display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c1))
							})
						})
					})

					Convey("Given a 'offset' parameter of 1", func() {
						commandLine.Account.Local.Args.Server = server.Name
						commandLine.Account.Local.Cert.Args.Account = account.Login
						args := []string{"-o", "1"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then it should NOT display the 1st certificate", func() {
								So(getOutput(), ShouldEqual, "Certificates:\n"+
									certInfoString(c2))
							})
						})
					})

					Convey("Given a 'sort' parameter of 'name-'", func() {
						commandLine.Account.Local.Args.Server = server.Name
						commandLine.Account.Local.Cert.Args.Account = account.Login
						args := []string{"-s", "name-"}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

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

func TestUpdateCertificate(t *testing.T) {

	Convey("Testing the certificate 'delete' command", t, func() {
		out = testFile()
		command := &certUpdate{}
		commandLine = options{}

		Convey("Given a gateway", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			cPk := writeFile(testhelpers.ClientKey)
			cCrt := writeFile(testhelpers.ClientCert)
			sPk := writeFile(testhelpers.LocalhostKey)
			sCrt := writeFile(testhelpers.LocalhostCert)

			Convey("Given a partner", func() {
				partner := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(partner).Run(), ShouldBeNil)

				Convey("When updating the certificate", func() {
					cert := &model.Crypto{
						OwnerType:   partner.TableName(),
						OwnerID:     partner.ID,
						Name:        "partner_cert",
						Certificate: testhelpers.LocalhostCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid partner, certificate & flags", func() {
						commandLine.Partner.Cert.Args.Partner = partner.Name
						args := []string{
							"-c", sCrt.Name(),
							cert.Name,
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then is should display a message saying the "+
								"cert was added", func() {
								So(getOutput(), ShouldEqual, "The certificate "+
									cert.Name+" was successfully updated.\n")
							})

							Convey("Then the cert should have been updated", func() {
								check := model.Crypto{
									ID:          cert.ID,
									OwnerType:   partner.TableName(),
									OwnerID:     partner.ID,
									Name:        "partner_cert",
									Certificate: testhelpers.LocalhostCert,
								}
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, check)
							})
						})
					})

					Convey("Given an invalid partner name", func() {
						commandLine.Partner.Cert.Args.Partner = "toto"
						args := []string{"-n", "partner_cert",
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
							cert.Name,
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then is should return an error", func() {
								So(err, ShouldBeError, "partner 'toto' not found")
							})

							Convey("Then the new cert should NOT have been changed", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})

					Convey("Given an invalid certificate name", func() {
						commandLine.Partner.Cert.Args.Partner = partner.Name
						args := []string{"-n", "partner_cert",
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
							"toto",
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then is should return an error", func() {
								So(err, ShouldBeError, "certificate 'toto' not found")
							})

							Convey("Then the new cert should NOT have been changed", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})
				})

				Convey("Given a partner account", func() {
					account := &model.RemoteAccount{
						RemoteAgentID: partner.ID,
						Login:         "rem_account",
						Password:      "password",
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					Convey("When updating the certificate", func() {
						cert := &model.Crypto{
							OwnerType:   account.TableName(),
							OwnerID:     account.ID,
							Name:        "account_cert",
							PrivateKey:  testhelpers.ClientKey,
							Certificate: testhelpers.ClientCert,
						}
						So(db.Insert(cert).Run(), ShouldBeNil)

						Convey("Given valid account, partner & flags", func() {
							commandLine.Account.Remote.Args.Partner = partner.Name
							commandLine.Account.Remote.Cert.Args.Account = account.Login
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								cert.Name,
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								So(command.Execute(params), ShouldBeNil)

								Convey("Then is should display a message saying "+
									"the cert was added", func() {
									So(getOutput(), ShouldEqual, "The certificate "+
										cert.Name+" was successfully updated.\n")
								})

								Convey("Then the cert should have been updated", func() {
									check := model.Crypto{
										ID:          cert.ID,
										OwnerType:   account.TableName(),
										OwnerID:     account.ID,
										Name:        "account_cert",
										PrivateKey:  testhelpers.ClientKey,
										Certificate: testhelpers.ClientCert,
									}
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, check)
								})
							})
						})

						Convey("Given an invalid partner name", func() {
							commandLine.Account.Remote.Args.Partner = "toto"
							commandLine.Account.Remote.Cert.Args.Account = account.Login
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								cert.Name,
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "partner 'toto' not found")
								})

								Convey("Then the cert should NOT have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, *cert)
								})
							})
						})

						Convey("Given an invalid account name", func() {
							commandLine.Account.Remote.Args.Partner = partner.Name
							commandLine.Account.Remote.Cert.Args.Account = "toto"
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								cert.Name,
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "no account 'toto' "+
										"found for partner "+partner.Name)
								})

								Convey("Then the cert should NOT have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, *cert)
								})
							})
						})

						Convey("Given an invalid certificate name", func() {
							commandLine.Account.Remote.Args.Partner = partner.Name
							commandLine.Account.Remote.Cert.Args.Account = account.Login
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								"toto",
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "certificate 'toto' not found")
								})

								Convey("Then the cert should NOT have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, *cert)
								})
							})
						})
					})
				})
			})

			Convey("Given a server", func() {
				server := &model.LocalAgent{
					Name:        "server",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:6666",
				}
				So(db.Insert(server).Run(), ShouldBeNil)

				Convey("When updating the certificate", func() {
					cert := &model.Crypto{
						OwnerType:   server.TableName(),
						OwnerID:     server.ID,
						Name:        "server_cert",
						PrivateKey:  testhelpers.LocalhostKey,
						Certificate: testhelpers.LocalhostCert,
					}
					So(db.Insert(cert).Run(), ShouldBeNil)

					Convey("Given valid server, certificate & flags", func() {
						commandLine.Server.Cert.Args.Server = server.Name
						args := []string{
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
							cert.Name,
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							So(command.Execute(params), ShouldBeNil)

							Convey("Then is should display a message saying "+
								"the cert was added", func() {
								So(getOutput(), ShouldEqual, "The certificate "+
									cert.Name+" was successfully updated.\n")
							})

							Convey("Then the cert should have been updated", func() {
								check := model.Crypto{
									ID:          cert.ID,
									OwnerType:   server.TableName(),
									OwnerID:     server.ID,
									Name:        "server_cert",
									PrivateKey:  testhelpers.LocalhostKey,
									Certificate: testhelpers.LocalhostCert,
								}
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, check)
							})
						})
					})

					Convey("Given an invalid server name", func() {
						commandLine.Server.Cert.Args.Server = "toto"
						args := []string{
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
							cert.Name,
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then is should return an error", func() {
								So(err, ShouldBeError, "server 'toto' not found")
							})

							Convey("Then the new cert should NOT have been changed", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})

					Convey("Given an invalid certificate name", func() {
						commandLine.Server.Cert.Args.Server = server.Name
						args := []string{
							"-p", sPk.Name(),
							"-c", sCrt.Name(),
							"toto",
						}

						Convey("When executing the command", func() {
							params, err := flags.ParseArgs(command, args)
							So(err, ShouldBeNil)
							err = command.Execute(params)

							Convey("Then is should return an error", func() {
								So(err, ShouldBeError, "certificate 'toto' not found")
							})

							Convey("Then the new cert should NOT have been changed", func() {
								var certs model.Cryptos
								So(db.Select(&certs).Run(), ShouldBeNil)
								So(certs, ShouldNotBeEmpty)
								So(certs, ShouldContain, *cert)
							})
						})
					})
				})

				Convey("Given a server account", func() {
					account := &model.LocalAccount{
						LocalAgentID: server.ID,
						Login:        "loc_account",
						PasswordHash: hash("password"),
					}
					So(db.Insert(account).Run(), ShouldBeNil)

					Convey("When updating the certificate", func() {
						cert := &model.Crypto{
							OwnerType:   account.TableName(),
							OwnerID:     account.ID,
							Name:        "account_cert",
							Certificate: testhelpers.ClientCert,
						}
						So(db.Insert(cert).Run(), ShouldBeNil)

						Convey("Given valid account, server & flags", func() {
							commandLine.Account.Local.Args.Server = server.Name
							commandLine.Account.Local.Cert.Args.Account = account.Login
							args := []string{
								"-c", cCrt.Name(),
								cert.Name,
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								So(command.Execute(params), ShouldBeNil)

								Convey("Then is should display a message saying "+
									"the cert was added", func() {
									So(getOutput(), ShouldEqual, "The certificate "+
										cert.Name+" was successfully updated.\n")
								})

								Convey("Then the cert should have been updated", func() {
									check := model.Crypto{
										ID:          cert.ID,
										OwnerType:   account.TableName(),
										OwnerID:     account.ID,
										Name:        "account_cert",
										Certificate: testhelpers.ClientCert,
									}
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, check)
								})
							})
						})

						Convey("Given an invalid server name", func() {
							commandLine.Account.Local.Args.Server = "toto"
							commandLine.Account.Local.Cert.Args.Account = account.Login
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								cert.Name,
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "server 'toto' not found")
								})

								Convey("Then the cert should NOT have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, *cert)
								})
							})
						})

						Convey("Given an invalid account name", func() {
							commandLine.Account.Local.Args.Server = server.Name
							commandLine.Account.Local.Cert.Args.Account = "toto"
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								cert.Name,
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "no account 'toto' "+
										"found for server "+server.Name)
								})

								Convey("Then the cert should NOT have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, *cert)
								})
							})
						})

						Convey("Given an invalid certificate name", func() {
							commandLine.Account.Local.Args.Server = server.Name
							commandLine.Account.Local.Cert.Args.Account = account.Login
							args := []string{
								"-p", cPk.Name(),
								"-c", cCrt.Name(),
								"toto",
							}

							Convey("When executing the command", func() {
								params, err := flags.ParseArgs(command, args)
								So(err, ShouldBeNil)
								err = command.Execute(params)

								Convey("Then is should return an error", func() {
									So(err, ShouldBeError, "certificate 'toto' not found")
								})

								Convey("Then the cert should NOT have been updated", func() {
									var certs model.Cryptos
									So(db.Select(&certs).Run(), ShouldBeNil)
									So(certs, ShouldNotBeEmpty)
									So(certs, ShouldContain, *cert)
								})
							})
						})
					})
				})
			})
		})
	})
}
