//+build old

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
	listener, err := makeDummyServer(testPK, testPBK, testLogin, testPassword)
	if err != nil {
		log.Fatal(err)
	}
	clientTestPort = uint16(listener.Addr().(*net.TCPAddr).Port)
}

func TestRequest(t *testing.T) {
	Convey("Given a SFTP client", t, func() {

		client := NewClient()
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
				SourceFile: "protocol_handler.go",
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
		client := &client{
			conf: &config.SftpProtoConfig{},
			Info: model.OutTransferInfo{
				Agent: &model.RemoteAgent{
					Address: fmt.Sprintf("localhost:%d", clientTestPort),
				},
				Account: &model.RemoteAccount{
					Login:    testLogin,
					Password: []byte(testPassword),
				},
				ServerCerts: []model.Cert{{
					PublicKey: testPBK,
				}},
			},
		}
		So(client.Connect(), ShouldBeNil)
		Reset(func() { _ = client.conn.Close() })

		srcFile := "client_test.src"
		content := []byte("client test transfer file content")
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
