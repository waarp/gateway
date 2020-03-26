package sftp

import (
	"io/ioutil"
	"log"
	"net"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	listener, err := makeDummyServer(testPK, testPBK, testLogin, testPassword)
	if err != nil {
		log.Fatal(err)
	}
	clientTestPort = uint16(listener.Addr().(*net.TCPAddr).Port)
}

func TestConnect(t *testing.T) {
	Convey("Given a SFTP client", t, func() {
		client := &Client{}

		Convey("Given a valid address", func() {
			client.conf = &config.SftpProtoConfig{
				Port:    clientTestPort,
				Address: "localhost",
			}

			Convey("When calling the `Connect` method", func() {
				err := client.Connect()

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then the connection should be open", func() {
						So(client.conn, ShouldNotBeNil)
						So(client.conn.Close(), ShouldBeNil)
					})
				})
			})
		})

		Convey("Given an incorrect address", func() {
			client.conf = &config.SftpProtoConfig{
				Port:    clientTestPort,
				Address: "255.255.255.255",
			}

			Convey("When calling the `Connect` method", func() {
				err := client.Connect()

				Convey("Then it should return an error", func() {
					So(err.Kind, ShouldEqual, model.KindTransfer)
					So(err.Cause.Code, ShouldEqual, model.TeConnection)

					Convey("Then the connection should NOT be open", func() {
						So(client.conn, ShouldBeNil)
					})
				})
			})
		})
	})
}

func TestAuthenticate(t *testing.T) {
	Convey("Given a SFTP client", t, func() {
		client := &Client{
			conf: &config.SftpProtoConfig{
				Port:    clientTestPort,
				Address: "localhost",
			},
		}
		So(client.Connect(), ShouldBeNil)
		Reset(func() { _ = client.conn.Close() })

		Convey("Given a valid SFTP configuration", func() {
			client.Info = model.OutTransferInfo{
				Account: &model.RemoteAccount{
					Login:    testLogin,
					Password: []byte("testPassword"),
				},
				ServerCerts: []model.Cert{{
					PublicKey: testPBK,
				}},
				ClientCerts: []model.Cert{{
					PrivateKey: []byte(rsaPK),
				}},
			}

			err := client.Authenticate()

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the SSH tunnel should be opened", func() {
					So(client.client, ShouldNotBeNil)
					So(client.client.Close(), ShouldBeNil)
				})
			})
		})

		SkipConvey("Given an incorrect SFTP configuration", func() {
			client.Info = model.OutTransferInfo{
				Account: &model.RemoteAccount{
					Login:    testLogin,
					Password: []byte("tutu"),
				},
				ServerCerts: []model.Cert{{
					PublicKey: testPBK,
				}},
			}

			err := client.Authenticate()

			Convey("Then it should return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the SSH tunnel NOT should be opened", func() {
					So(client.client, ShouldBeNil)
				})
			})
		})
	})
}

func TestRequest(t *testing.T) {
	Convey("Given a SFTP client", t, func() {
		client := &Client{
			conf: &config.SftpProtoConfig{
				Port:    clientTestPort,
				Address: "localhost",
			},
		}
		So(client.Connect(), ShouldBeNil)
		Reset(func() { _ = client.conn.Close() })

		client.Info = model.OutTransferInfo{
			Account: &model.RemoteAccount{
				Login:    testLogin,
				Password: []byte(testPassword),
			},
			ServerCerts: []model.Cert{{
				PublicKey: testPBK,
			}},
		}

		So(client.Authenticate(), ShouldBeNil)
		Reset(func() { _ = client.client.Close() })

		Convey("Given a valid out file transfer", func() {
			client.Info.Transfer = &model.Transfer{
				DestPath: "client_test.dst",
			}
			client.Info.Rule = &model.Rule{
				IsSend: true,
				Path:   ".",
			}

			err := client.Request()
			Reset(func() { _ = os.Remove(client.Info.Transfer.DestPath) })

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the file stream should be open", func() {
					So(client.remoteFile, ShouldNotBeNil)
					So(client.remoteFile.Close(), ShouldBeNil)
				})
			})
		})

		Convey("Given a valid in file transfer", func() {
			client.Info.Transfer = &model.Transfer{
				SourcePath: "client.go",
			}
			client.Info.Rule = &model.Rule{
				IsSend: false,
			}

			err := client.Request()

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the file stream should be open", func() {
					So(client.remoteFile, ShouldNotBeNil)
					So(client.remoteFile.Close(), ShouldBeNil)
				})
			})
		})

		Convey("Given an invalid in file transfer", func() {
			client.Info.Transfer = &model.Transfer{
				SourcePath: "unknown.file",
			}
			client.Info.Rule = &model.Rule{
				IsSend: false,
			}

			err := client.Request()

			Convey("Then it should return an error", func() {
				So(err, ShouldResemble, model.NewPipelineError(model.TeFileNotFound,
					"Target file does not exist"))

				Convey("Then the file stream should NOT be open", func() {
					So(client.remoteFile, ShouldBeNil)
				})
			})
		})
	})
}

func TestData(t *testing.T) {
	Convey("Given a SFTP client", t, func() {
		client := &Client{
			conf: &config.SftpProtoConfig{
				Port:    clientTestPort,
				Address: "localhost",
			},
		}
		So(client.Connect(), ShouldBeNil)
		Reset(func() { _ = client.conn.Close() })

		client.Info = model.OutTransferInfo{
			Account: &model.RemoteAccount{
				Login:    testLogin,
				Password: []byte(testPassword),
			},
			ServerCerts: []model.Cert{{
				PublicKey: testPBK,
			}},
		}

		So(client.Authenticate(), ShouldBeNil)
		Reset(func() { _ = client.client.Close() })

		Convey("Given a valid out file transfer", func() {
			client.Info.Transfer = &model.Transfer{
				DestPath:   "client_test_in.dst",
				SourcePath: "client.go",
			}
			client.Info.Rule = &model.Rule{
				IsSend: true,
				Path:   ".",
			}

			So(client.Request(), ShouldBeNil)
			Reset(func() { _ = os.Remove(client.Info.Transfer.DestPath) })

			file, err := os.Open(client.Info.Transfer.SourcePath)
			So(err, ShouldBeNil)

			err = client.Data(file)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the file should have been copied", func() {
					src, err := ioutil.ReadFile(client.Info.Transfer.SourcePath)
					So(err, ShouldBeNil)
					dst, err := ioutil.ReadFile(client.Info.Transfer.DestPath)
					So(err, ShouldBeNil)

					So(src, ShouldResemble, dst)
				})
			})
		})

		Convey("Given a valid in file transfer", func() {
			client.Info.Transfer = &model.Transfer{
				DestPath:   "client_test_out.dst",
				SourcePath: "client.go",
			}
			client.Info.Rule = &model.Rule{
				IsSend: false,
			}

			So(client.Request(), ShouldBeNil)

			file, err := os.Create(client.Info.Transfer.DestPath)
			So(err, ShouldBeNil)
			Reset(func() { _ = os.Remove(client.Info.Transfer.DestPath) })

			err = client.Data(file)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the file should have been copied", func() {
					src, err := ioutil.ReadFile(client.Info.Transfer.SourcePath)
					So(err, ShouldBeNil)
					dst, err := ioutil.ReadFile(client.Info.Transfer.DestPath)
					So(err, ShouldBeNil)

					So(src, ShouldResemble, dst)
				})
			})
		})
	})
}
