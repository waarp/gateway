package sftp

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	listener, err := makeDummyServer(rsaPK, rsaPBK, testLogin, testPassword)
	if err != nil {
		log.Fatal(err)
	}
	clientTestPort = uint16(listener.Addr().(*net.TCPAddr).Port)
}

func TestConnect(t *testing.T) {
	Convey("Given a SFTP client", t, func() {
		client := &Client{}

		Convey("Given a valid address", func() {
			client.Info = model.OutTransferInfo{
				Agent: &model.RemoteAgent{
					Address: fmt.Sprintf("localhost:%d", clientTestPort),
				},
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
			client.Info = model.OutTransferInfo{
				Agent: &model.RemoteAgent{
					Address: fmt.Sprintf("255.255.255.255:%d", clientTestPort),
				},
			}

			Convey("When calling the `Connect` method", func() {
				err := client.Connect()

				Convey("Then it should return an error", func() {
					te, ok := err.(types.TransferError)
					So(ok, ShouldBeTrue)
					So(te.Code, ShouldEqual, types.TeConnection)

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
			conf: &config.SftpProtoConfig{},
			Info: model.OutTransferInfo{
				Agent: &model.RemoteAgent{
					Address: fmt.Sprintf("localhost:%d", clientTestPort),
				},
			},
		}
		So(client.Connect(), ShouldBeNil)
		Reset(func() { _ = client.conn.Close() })

		Convey("Given a valid SFTP configuration", func() {
			client.Info.Account = &model.RemoteAccount{
				Login:    testLogin,
				Password: []byte("testPassword"),
			}
			client.Info.ServerCryptos = []model.Crypto{{
				SSHPublicKey: rsaPBK,
			}}
			client.Info.ClientCryptos = []model.Crypto{{
				PrivateKey: rsaPK,
			}}

			err := client.Authenticate()

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the SSH tunnel should be opened", func() {
					So(client.client, ShouldNotBeNil)
					So(client.client.Close(), ShouldBeNil)
				})
			})
		})

		Convey("Given an incorrect SFTP configuration", func() {
			client.Info.Account = &model.RemoteAccount{
				Login:    testLogin,
				Password: []byte("tutu"),
			}
			client.Info.ServerCryptos = []model.Crypto{{
				SSHPublicKey: rsaPBK,
			}}

			err := client.Authenticate()

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)

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
			conf: &config.SftpProtoConfig{},
			Info: model.OutTransferInfo{
				Agent: &model.RemoteAgent{
					Address: fmt.Sprintf("localhost:%d", clientTestPort),
				},
				Account: &model.RemoteAccount{
					Login:    testLogin,
					Password: []byte(testPassword),
				},
				ServerCryptos: []model.Crypto{{
					SSHPublicKey: rsaPBK,
				}},
			},
		}
		So(client.Connect(), ShouldBeNil)
		defer func() { _ = client.conn.Close() }()

		So(client.Authenticate(), ShouldBeNil)
		defer func() { _ = client.client.Close() }()

		Convey("Given a valid out file transfer", func() {
			client.Info.Transfer = &model.Transfer{
				DestFile: "client_test.dst",
			}
			client.Info.Rule = &model.Rule{
				IsSend: true,
				Path:   ".",
			}

			err := client.Request()
			Reset(func() { _ = os.Remove(client.Info.Transfer.DestFile) })

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
				SourceFile: "client.go",
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
				SourceFile: "unknown.file",
			}
			client.Info.Rule = &model.Rule{
				IsSend: false,
			}

			err := client.Request()

			Convey("Then it should return an error", func() {
				So(err, ShouldResemble, types.NewTransferError(types.TeFileNotFound,
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
			conf: &config.SftpProtoConfig{},
			Info: model.OutTransferInfo{
				Agent: &model.RemoteAgent{
					Address: fmt.Sprintf("localhost:%d", clientTestPort),
				},
				Account: &model.RemoteAccount{
					Login:    testLogin,
					Password: []byte(testPassword),
				},
				ServerCryptos: []model.Crypto{{
					SSHPublicKey: rsaPBK,
				}},
			},
		}
		So(client.Connect(), ShouldBeNil)
		Reset(func() { _ = client.conn.Close() })

		srcFile := "client_test.src"
		content := []byte("Client test transfer file content")
		So(ioutil.WriteFile(srcFile, content, 0600), ShouldBeNil)
		Reset(func() { _ = os.Remove(client.Info.Transfer.SourceFile) })

		So(client.Authenticate(), ShouldBeNil)
		Reset(func() { _ = client.client.Close() })

		Convey("Given a valid out file transfer", func() {
			client.Info.Transfer = &model.Transfer{
				SourceFile: srcFile,
				DestFile:   "client_test_out.dst",
			}

			client.Info.Rule = &model.Rule{
				IsSend: true,
			}
			So(client.Request(), ShouldBeNil)
			Reset(func() {
				_ = client.Close(nil)
				_ = os.Remove(client.Info.Transfer.DestFile)
			})

			Convey("Given that the requests succeeded", func() {
				file, err := os.Open(client.Info.Transfer.SourceFile)
				So(err, ShouldBeNil)
				Reset(func() { _ = file.Close() })

				err = client.Data(file)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then the file should have been copied", func() {
						dst, err := ioutil.ReadFile(client.Info.Transfer.DestFile)
						So(err, ShouldBeNil)

						So(dst, ShouldResemble, content)
					})
				})
			})
		})

		Convey("Given a valid in file transfer", func() {
			client.Info.Transfer = &model.Transfer{
				DestFile:   "client_test_in.dst",
				SourceFile: srcFile,
			}
			client.Info.Rule = &model.Rule{
				IsSend: false,
			}

			So(client.Request(), ShouldBeNil)
			Reset(func() {
				_ = client.Close(nil)
				_ = os.Remove(client.Info.Transfer.DestFile)
			})

			Convey("Given that the request succeeded", func() {
				file, err := os.Create(client.Info.Transfer.DestFile)
				So(err, ShouldBeNil)
				Reset(func() { _ = file.Close() })

				err = client.Data(file)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then the file should have been copied", func() {
						dst, err := ioutil.ReadFile(client.Info.Transfer.DestFile)
						So(err, ShouldBeNil)

						So(dst, ShouldResemble, content)
					})
				})
			})
		})
	})
}
