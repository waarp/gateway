package main

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func writeFile(content string) *os.File {
	file := testFile()
	_, err := file.WriteString(content)
	So(err, ShouldBeNil)
	return file
}

func TestGetCertificate(t *testing.T) {

	Convey("Testing the certificate 'get' command", t, func() {
		out = testFile()
		command := &certGetCommand{}

		Convey("Given a gateway with 1 certificate", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			owner := model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}

			err := db.Create(&owner)
			So(err, ShouldBeNil)

			cert := model.Cert{
				OwnerType:   owner.TableName(),
				OwnerID:     owner.ID,
				Name:        "cert",
				PrivateKey:  []byte("private_key"),
				PublicKey:   []byte("public_key"),
				Certificate: []byte("certificate_content"),
			}

			err = db.Create(&cert)
			So(err, ShouldBeNil)

			Convey("Given a valid server ID", func() {
				id := fmt.Sprint(cert.ID)

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the certificate's info", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Certificate n°1:\n"+
							"        Name: "+cert.Name+"\n"+
							"        Type: "+cert.OwnerType+"\n"+
							"       Owner: "+fmt.Sprint(cert.OwnerID)+"\n"+
							" Private key: "+string(cert.PrivateKey)+"\n"+
							"  Public key: "+string(cert.PublicKey)+"\n"+
							"     Content: "+fmt.Sprint(cert.Certificate)+"\n",
						)
					})
				})
			})

			Convey("Given an invalid server ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+rest.CertificatesPath+
							"/1000' does not exist")

					})
				})
			})
		})
	})
}

func TestAddCertificate(t *testing.T) {

	Convey("Testing the cert 'add' command", t, func() {
		out = testFile()
		command := &certAddCommand{}

		Convey("Given a gateway", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			owner := model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}

			err := db.Create(&owner)
			So(err, ShouldBeNil)

			Convey("When adding a new certificate", func() {
				prK := writeFile("private_key")
				puK := writeFile("public_key")
				crt := writeFile("certificate")

				command.Name = "new_cert"
				command.Type = owner.TableName()
				command.Owner = owner.ID
				command.PrivateKey = prK.Name()
				command.PublicKey = puK.Name()
				command.Certificate = crt.Name()

				Convey("Given valid parameters", func() {

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then is should display a message saying the cert was added", func() {
							_, err = out.Seek(0, 0)
							So(err, ShouldBeNil)
							cont, err := ioutil.ReadAll(out)
							So(err, ShouldBeNil)
							So(string(cont), ShouldEqual, "The certificate '"+command.Name+
								"' was successfully added. It can be consulted at "+
								"the address: "+gw.URL+admin.APIPath+
								rest.CertificatesPath+"/1\n")
						})

						Convey("Then the new certificate should have been added", func() {
							cert := model.Cert{
								OwnerType:   command.Type,
								OwnerID:     command.Owner,
								Name:        command.Name,
								PrivateKey:  []byte("private_key"),
								PublicKey:   []byte("public_key"),
								Certificate: []byte("certificate"),
							}
							exists, err := db.Exists(&cert)
							So(err, ShouldBeNil)
							So(exists, ShouldBeTrue)
						})
					})
				})

				Convey("Given an invalid 'type'", func() {
					command.Type = "invalid"

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute(nil)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "400 - Invalid request: "+
								"The certificate's owner type must be one of "+
								"[local_agents remote_agents local_accounts remote_accounts]")
						})
					})
				})

				Convey("Given an invalid 'owner'", func() {
					command.Owner = 1000

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute(nil)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "400 - Invalid request: "+
								"No remote_agents found with ID '1000'")
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
		command := &certDeleteCommand{}

		Convey("Given a gateway with 1 certificate", func() {

			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			owner := model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err := db.Create(&owner)
			So(err, ShouldBeNil)

			cert := model.Cert{
				OwnerType:   owner.TableName(),
				OwnerID:     owner.ID,
				Name:        "cert",
				PrivateKey:  []byte("private_key"),
				PublicKey:   []byte("public_key"),
				Certificate: []byte("certificate_content"),
			}
			err = db.Create(&cert)
			So(err, ShouldBeNil)

			Convey("Given a valid cert ID", func() {
				id := fmt.Sprint(cert.ID)

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then is should display a message saying the certificate was deleted", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "The certificate n°"+id+
							" was successfully deleted from the database\n")
					})

					Convey("Then the certificate should have been removed", func() {
						exists, err := db.Exists(&cert)
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
					})
				})
			})

			Convey("Given an invalid ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+rest.CertificatesPath+
							"/1000' does not exist")
					})

					Convey("Then the cert should still exist", func() {
						exists, err := db.Exists(&cert)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})
		})
	})
}

func TestListCertificate(t *testing.T) {

	Convey("Testing the certificate 'list' command", t, func() {
		out = testFile()
		command := &certListCommand{}
		_, err := flags.ParseArgs(command, []string{"waarp_gateway"})
		So(err, ShouldBeNil)

		Convey("Given a gateway with 2 certificates", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			owner1 := model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err := db.Create(&owner1)
			So(err, ShouldBeNil)

			owner2 := model.LocalAgent{
				Name:        "local_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err = db.Create(&owner2)
			So(err, ShouldBeNil)

			cert1 := model.Cert{
				OwnerType:   owner1.TableName(),
				OwnerID:     owner1.ID,
				Name:        "cert1",
				PrivateKey:  []byte("private_key_1"),
				PublicKey:   []byte("public_key_1"),
				Certificate: []byte("certificate_content_1"),
			}
			err = db.Create(&cert1)
			So(err, ShouldBeNil)

			cert2 := model.Cert{
				OwnerType:   owner2.TableName(),
				OwnerID:     owner2.ID,
				Name:        "cert2",
				PrivateKey:  []byte("private_key_2"),
				PublicKey:   []byte("public_key_2"),
				Certificate: []byte("certificate_content_2"),
			}
			err = db.Create(&cert2)
			So(err, ShouldBeNil)

			Convey("Given no parameters", func() {

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the certificates' info", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Certificates:\n"+
							"Certificate n°1:\n"+
							"        Name: "+cert1.Name+"\n"+
							"        Type: "+cert1.OwnerType+"\n"+
							"       Owner: "+fmt.Sprint(cert1.OwnerID)+"\n"+
							" Private key: "+string(cert1.PrivateKey)+"\n"+
							"  Public key: "+string(cert1.PublicKey)+"\n"+
							"     Content: "+fmt.Sprint(cert1.Certificate)+"\n"+
							"Certificate n°2:\n"+
							"        Name: "+cert2.Name+"\n"+
							"        Type: "+cert2.OwnerType+"\n"+
							"       Owner: "+fmt.Sprint(cert2.OwnerID)+"\n"+
							" Private key: "+string(cert2.PrivateKey)+"\n"+
							"  Public key: "+string(cert2.PublicKey)+"\n"+
							"     Content: "+fmt.Sprint(cert2.Certificate)+"\n",
						)
					})
				})
			})

			Convey("Given a 'limit' parameter of 1", func() {
				command.Limit = 1

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display only 1 certificate's info", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Certificates:\n"+
							"Certificate n°1:\n"+
							"        Name: "+cert1.Name+"\n"+
							"        Type: "+cert1.OwnerType+"\n"+
							"       Owner: "+fmt.Sprint(cert1.OwnerID)+"\n"+
							" Private key: "+string(cert1.PrivateKey)+"\n"+
							"  Public key: "+string(cert1.PublicKey)+"\n"+
							"     Content: "+fmt.Sprint(cert1.Certificate)+"\n",
						)
					})
				})
			})

			Convey("Given an 'offset' parameter of 1", func() {
				command.Offset = 1

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should NOT display the 1st certificate", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Certificates:\n"+
							"Certificate n°2:\n"+
							"        Name: "+cert2.Name+"\n"+
							"        Type: "+cert2.OwnerType+"\n"+
							"       Owner: "+fmt.Sprint(cert2.OwnerID)+"\n"+
							" Private key: "+string(cert2.PrivateKey)+"\n"+
							"  Public key: "+string(cert2.PublicKey)+"\n"+
							"     Content: "+fmt.Sprint(cert2.Certificate)+"\n",
						)
					})
				})
			})

			Convey("Given the 'desc' flag is set", func() {
				command.DescOrder = true

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the certificates' info in reverse", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Certificates:\n"+
							"Certificate n°2:\n"+
							"        Name: "+cert2.Name+"\n"+
							"        Type: "+cert2.OwnerType+"\n"+
							"       Owner: "+fmt.Sprint(cert2.OwnerID)+"\n"+
							" Private key: "+string(cert2.PrivateKey)+"\n"+
							"  Public key: "+string(cert2.PublicKey)+"\n"+
							"     Content: "+fmt.Sprint(cert2.Certificate)+"\n"+
							"Certificate n°1:\n"+
							"        Name: "+cert1.Name+"\n"+
							"        Type: "+cert1.OwnerType+"\n"+
							"       Owner: "+fmt.Sprint(cert1.OwnerID)+"\n"+
							" Private key: "+string(cert1.PrivateKey)+"\n"+
							"  Public key: "+string(cert1.PublicKey)+"\n"+
							"     Content: "+fmt.Sprint(cert1.Certificate)+"\n",
						)
					})
				})
			})

			Convey("Given a 'partner' parameter", func() {
				command.Partner = []uint64{owner1.ID}

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should only display the partner's certificates", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Certificates:\n"+
							"Certificate n°1:\n"+
							"        Name: "+cert1.Name+"\n"+
							"        Type: "+cert1.OwnerType+"\n"+
							"       Owner: "+fmt.Sprint(cert1.OwnerID)+"\n"+
							" Private key: "+string(cert1.PrivateKey)+"\n"+
							"  Public key: "+string(cert1.PublicKey)+"\n"+
							"     Content: "+fmt.Sprint(cert1.Certificate)+"\n",
						)
					})
				})
			})

			Convey("Given a 'server' parameter", func() {
				command.Server = []uint64{owner2.ID}

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the server's certificates", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Certificates:\n"+
							"Certificate n°2:\n"+
							"        Name: "+cert2.Name+"\n"+
							"        Type: "+cert2.OwnerType+"\n"+
							"       Owner: "+fmt.Sprint(cert2.OwnerID)+"\n"+
							" Private key: "+string(cert2.PrivateKey)+"\n"+
							"  Public key: "+string(cert2.PublicKey)+"\n"+
							"     Content: "+fmt.Sprint(cert2.Certificate)+"\n",
						)
					})
				})
			})
		})
	})
}

func TestUpdateCertificate(t *testing.T) {

	Convey("Testing the certificate 'delete' command", t, func() {
		out = testFile()
		command := &certUpdateCommand{}

		Convey("Given a gateway with 1 certificate", func() {

			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			owner := model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err := db.Create(&owner)
			So(err, ShouldBeNil)

			cert := model.Cert{
				OwnerType:   owner.TableName(),
				OwnerID:     owner.ID,
				Name:        "cert",
				PrivateKey:  []byte("private_key"),
				PublicKey:   []byte("public_key"),
				Certificate: []byte("certificate_content"),
			}
			err = db.Create(&cert)
			So(err, ShouldBeNil)

			Convey("Given a valid certificate ID", func() {
				id := fmt.Sprint(owner.ID)

				prK := writeFile("new_private_key")
				puK := writeFile("new_public_key")
				crt := writeFile("new_certificate")

				command.Name = "new_cert"
				command.Type = owner.TableName()
				command.Owner = owner.ID
				command.PrivateKey = prK.Name()
				command.PublicKey = puK.Name()
				command.Certificate = crt.Name()

				Convey("Given all valid flags", func() {

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute([]string{id})

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then is should display a message saying the certificate was updated", func() {
							_, err = out.Seek(0, 0)
							So(err, ShouldBeNil)
							cont, err := ioutil.ReadAll(out)
							So(err, ShouldBeNil)
							So(string(cont), ShouldEqual, "The certificate n°"+id+
								" was successfully updated\n")
						})

						Convey("Then the old certificate should have been removed", func() {
							exists, err := db.Exists(&cert)
							So(err, ShouldBeNil)
							So(exists, ShouldBeFalse)
						})

						Convey("Then the new certificate should exist", func() {
							newCert := model.Cert{
								ID:          cert.ID,
								OwnerType:   command.Type,
								OwnerID:     command.Owner,
								Name:        command.Name,
								PrivateKey:  []byte("new_private_key"),
								PublicKey:   []byte("new_public_key"),
								Certificate: []byte("new_certificate"),
							}
							exists, err := db.Exists(&newCert)
							So(err, ShouldBeNil)
							So(exists, ShouldBeTrue)
						})
					})
				})

				Convey("Given an invalid 'type'", func() {
					command.Type = "invalid"

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute([]string{id})

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "400 - Invalid request: "+
								"The certificate's owner type must be one of "+
								"[local_agents remote_agents local_accounts remote_accounts]")
						})
					})
				})

				Convey("Given an invalid 'owner'", func() {
					command.Owner = 1000

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute([]string{id})

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "400 - Invalid request: "+
								"No remote_agents found with ID '1000'")
						})
					})
				})
			})

			Convey("Given an invalid ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+rest.CertificatesPath+
							"/1000' does not exist")
					})

					Convey("Then the certificate should stay unchanged", func() {
						exists, err := db.Exists(&cert)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})
		})
	})
}
