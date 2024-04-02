package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"

	. "code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const transferURI = "http://localhost:8080/api/transfers"

//nolint:maintidx //the function is fine as is
func TestAddTransfer(t *testing.T) {
	Convey("Testing the transfer add handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_transfer_add_test")
		db := database.TestDatabase(c)
		handler := addTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 partner, 1 certificate & 1 account", func() {
			client := &model.Client{Name: "test_client", Protocol: testProto1}
			So(db.Insert(client).Run(), ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:     "remote",
				Protocol: client.Protocol,
				Address:  "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "toto",
				Password:      "sesame",
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			push := model.Rule{Name: "push", IsSend: true, Path: "/push"}
			So(db.Insert(&push).Run(), ShouldBeNil)

			Convey("Given a valid new transfer", func() {
				body := strings.NewReader(`{
					"rule": "push",
					"client": "test_client",
					"partner": "remote",
					"account": "toto",
					"isSend": true,
					"file": "src_dir/test.file",
					"output": "dst_dir/test.file",
					"start": "2023-01-01T01:00:00+00:00",
					"transferInfo": { "key1":"val1", "key2": 2, "key3": true }
				}`)

				Convey("When calling the handler", func() {
					r, err := http.NewRequest(http.MethodPost, transferURI, body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeBlank)
					})

					Convey("Then it should return a code 201", func() {
						So(w.Code, ShouldEqual, http.StatusCreated)
					})

					Convey("Then the 'Location' header should contain the URI "+
						"of the new transfer", func() {
						location := w.Header().Get("Location")
						So(location, ShouldStartWith, transferURI)
					})

					Convey("Then the new transfer should be inserted in "+
						"the database", func() {
						var transfers model.Transfers

						So(db.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldHaveLength, 1)

						So(transfers[0].ID, ShouldEqual, 1)
						So(transfers[0].ClientID.Int64, ShouldEqual, client.ID)
						So(transfers[0].RemoteTransferID, ShouldNotBeBlank)
						So(transfers[0].RuleID, ShouldEqual, push.ID)
						So(transfers[0].RemoteAccountID.Int64, ShouldEqual, account.ID)
						So(transfers[0].SrcFilename, ShouldEqual, "src_dir/test.file")
						So(transfers[0].DestFilename, ShouldEqual, "dst_dir/test.file")
						So(transfers[0].LocalPath.String(), ShouldBeBlank)
						So(transfers[0].RemotePath, ShouldBeBlank)
						So(transfers[0].Filesize, ShouldEqual, model.UnknownSize)
						So(transfers[0].Start.Equal(time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)), ShouldBeTrue)
						So(transfers[0].Step, ShouldEqual, types.StepNone)
						So(transfers[0].Status, ShouldEqual, types.StatusPlanned)
						So(transfers[0].Owner, ShouldEqual, conf.GlobalConfig.GatewayName)
						So(transfers[0].Progress, ShouldEqual, 0)
						So(transfers[0].TaskNumber, ShouldEqual, 0)
						So(transfers[0].ErrCode, ShouldBeZeroValue)
						So(transfers[0].ErrDetails, ShouldBeBlank)

						info, err := transfers[0].GetTransferInfo(db)
						So(err, ShouldBeNil)
						So(info, ShouldResemble, map[string]any{
							"key1": "val1", "key2": float64(2), "key3": true,
						})
					})
				})
			})

			Convey("Given that the client wasn't specified", func() {
				body := strings.NewReader(`{
					"rule": "push",
					"partner": "remote",
					"account": "toto",
					"isSend": true,
					"file": "src_dir/test.file",
					"output": "/dst_dir/test.file",
					"start": "2023-01-01T01:00:00+00:00",
					"transferInfo": { "key1":"val1", "key2": 2, "key3": true }
				}`)

				Convey("When calling the handler", func() {
					r, err := http.NewRequest(http.MethodPost, transferURI, body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeBlank)
					})

					Convey("Then it should return a code 201", func() {
						So(w.Code, ShouldEqual, http.StatusCreated)
					})

					Convey("Then the 'Location' header should contain the URI "+
						"of the new transfer", func() {
						location := w.Header().Get("Location")
						So(location, ShouldStartWith, transferURI)
					})

					Convey("Then the new transfer should be inserted in "+
						"the database", func() {
						var transfers model.Transfers
						So(db.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldHaveLength, 1)

						So(transfers[0].ID, ShouldEqual, 1)
						So(transfers[0].RemoteTransferID, ShouldNotBeBlank)
						So(transfers[0].RuleID, ShouldEqual, push.ID)
						So(transfers[0].ClientID.Int64, ShouldEqual, client.ID)
						So(transfers[0].RemoteAccountID.Int64, ShouldEqual, account.ID)
						So(transfers[0].SrcFilename, ShouldEqual, "src_dir/test.file")
						So(transfers[0].DestFilename, ShouldEqual, "/dst_dir/test.file")
						So(transfers[0].Filesize, ShouldEqual, model.UnknownSize)
						So(transfers[0].Start.Equal(time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)), ShouldBeTrue)
						So(transfers[0].Step, ShouldEqual, types.StepNone)
						So(transfers[0].Status, ShouldEqual, types.StatusPlanned)
						So(transfers[0].Owner, ShouldEqual, conf.GlobalConfig.GatewayName)
						So(transfers[0].Progress, ShouldEqual, 0)
						So(transfers[0].TaskNumber, ShouldEqual, 0)
						So(transfers[0].ErrCode, ShouldBeZeroValue)
						So(transfers[0].ErrDetails, ShouldBeZeroValue)

						info, err := transfers[0].GetTransferInfo(db)
						So(err, ShouldBeNil)
						So(info, ShouldResemble, map[string]any{
							"key1": "val1", "key2": float64(2), "key3": true,
						})
					})
				})
			})

			Convey("Given a new transfer with an invalid rule name", func() {
				body := strings.NewReader(`{
					"rule": "tata",
					"client": "test_client",
					"partner": "remote",
					"account": "toto",
					"isSend": true,
					"file": "test.file"
				}`)

				Convey("When calling the handler", func() {
					r, err := http.NewRequest(http.MethodPost, transferURI, body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then it should return a code 400", func() {
						So(w.Code, ShouldEqual, http.StatusBadRequest)
					})

					Convey("Then the response body should say the partner is invalid", func() {
						So(w.Body.String(), ShouldEqual, "no rule 'tata' found\n")
					})
				})
			})

			Convey("Given a new transfer with an invalid partner name", func() {
				body := strings.NewReader(`{
					"rule": "push",
					"client": "test_client",
					"partner": "tata",
					"account": "toto",
					"isSend": true,
					"file": "test.file"
				}`)

				Convey("When calling the handler", func() {
					r, err := http.NewRequest(http.MethodPost, "", body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then it should return a code 400", func() {
						So(w.Code, ShouldEqual, http.StatusBadRequest)
					})

					Convey("Then the response body should say the partner is invalid", func() {
						So(w.Body.String(), ShouldEqual, "no partner 'tata' found\n")
					})
				})
			})

			Convey("Given a new transfer with an invalid account name", func() {
				body := strings.NewReader(`{
					"rule": "push",
					"client": "test_client",
					"partner": "remote",
					"account": "tata",
					"isSend": true,
					"file": "test.file"
				}`)

				Convey("When calling the handler", func() {
					r, err := http.NewRequest(http.MethodPost, "", body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then it should return a code 400", func() {
						So(w.Code, ShouldEqual, http.StatusBadRequest)
					})

					Convey("Then the response body should say the partner is invalid", func() {
						So(w.Body.String(), ShouldEqual, "no account 'tata' found "+
							"for partner "+partner.Name+"\n")
					})
				})
			})

			Convey("Given the transfer direction is missing", func() {
				body := strings.NewReader(`{
					"rule": "push",
					"client": "test_client",
					"partner": "remote",
					"account": "toto",
					"sourcePath": "file.src",
					"destPath": "file.dst"
				}`)

				Convey("When calling the handler", func() {
					//nolint:noctx //this is a test
					r, err := http.NewRequest(http.MethodPost, "", body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then it should return a code 400", func() {
						So(w.Code, ShouldEqual, http.StatusBadRequest)
					})

					Convey("Then the response body should say the direction is missing", func() {
						So(w.Body.String(), ShouldEqual, "the transfer direction "+
							"(isSend) is missing\n")
					})
				})
			})
		})
	})
}

func TestGetTransfer(t *testing.T) {
	Convey("Testing the transfer get handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_transfer_get_test")
		db := database.TestDatabase(c)
		handler := getTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 transfer", func() {
			client := &model.Client{Name: "test_client", Protocol: testProto1}
			So(db.Insert(client).Run(), ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:     "partner",
				Protocol: client.Protocol,
				Address:  "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "toto",
				Password:      "titi",
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			push := &model.Rule{Name: "push", IsSend: true, Path: "/push"}
			So(db.Insert(push).Run(), ShouldBeNil)

			// add a transfer from another gateway
			owner := conf.GlobalConfig.GatewayName
			conf.GlobalConfig.GatewayName = "foobar"
			other := &model.Transfer{
				RuleID:          push.ID,
				ClientID:        utils.NewNullInt64(client.ID),
				RemoteAccountID: utils.NewNullInt64(account.ID),
				SrcFilename:     "/source/file1.test",
				DestFilename:    "/dest/file1.test",
				Start:           time.Date(2021, 1, 1, 1, 0, 0, 0, time.Local),
			}
			So(db.Insert(other).Run(), ShouldBeNil)

			conf.GlobalConfig.GatewayName = owner

			trans := &model.Transfer{
				RuleID:          push.ID,
				ClientID:        utils.NewNullInt64(client.ID),
				RemoteAccountID: utils.NewNullInt64(account.ID),
				SrcFilename:     "/source/file2.test",
				DestFilename:    "/dest/file2.test",
				Start:           time.Date(2021, 1, 1, 1, 0, 0, 0, time.Local),
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			infos := map[string]any{"key1": "val1", "key2": 2}
			So(trans.SetTransferInfo(db, infos), ShouldBeNil)

			Convey("Given a request with the valid transfer ID parameter", func() {
				id := utils.FormatInt(trans.ID)
				uri := path.Join(transferURI, id)
				req, err := http.NewRequest(http.MethodGet, uri, nil)
				So(err, ShouldBeNil)

				req = mux.SetURLVars(req, map[string]string{"transfer": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested transfer "+
						"in JSON format", func() {
						jsonObj := fromTransfer(db, trans)
						exp, err := json.Marshal(jsonObj)
						So(err, ShouldBeNil)

						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with an invalid transfer ID parameter", func() {
				uri := path.Join(transferURI, "1000")
				r, err := http.NewRequest(http.MethodGet, uri, nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"transfer": "1000"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}

func TestListTransfer(t *testing.T) {
	Convey("Testing the transfer list handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_transfer_list_test")
		db := database.TestDatabase(c)
		handler := listTransfers(logger, db)
		w := httptest.NewRecorder()

		expected := map[string][]OutTransfer{}

		Convey("Given a database with 2 transfer", func() {
			cli1 := &model.Client{Name: "test_client1", Protocol: testProto1}
			So(db.Insert(cli1).Run(), ShouldBeNil)

			cli2 := &model.Client{Name: "test_client2", Protocol: testProto2}
			So(db.Insert(cli2).Run(), ShouldBeNil)

			p1 := &model.RemoteAgent{
				Name:     "part1",
				Protocol: cli1.Protocol,
				Address:  "localhost:1",
			}
			So(db.Insert(p1).Run(), ShouldBeNil)

			p2 := &model.RemoteAgent{
				Name:     "part2",
				Protocol: cli2.Protocol,
				Address:  "localhost:2",
			}
			So(db.Insert(p2).Run(), ShouldBeNil)

			a1 := &model.RemoteAccount{
				RemoteAgentID: p1.ID,
				Login:         "toto",
				Password:      "titi",
			}
			So(db.Insert(a1).Run(), ShouldBeNil)

			a2 := &model.RemoteAccount{
				RemoteAgentID: p2.ID,
				Login:         "toto",
				Password:      "titi",
			}
			So(db.Insert(a2).Run(), ShouldBeNil)

			r1 := &model.Rule{Name: "rule1", IsSend: false, Path: "path1"}
			So(db.Insert(r1).Run(), ShouldBeNil)

			r2 := &model.Rule{Name: "rule2", IsSend: false, Path: "path2"}
			So(db.Insert(r2).Run(), ShouldBeNil)

			t1 := &model.Transfer{
				RuleID:          r1.ID,
				ClientID:        utils.NewNullInt64(cli1.ID),
				RemoteAccountID: utils.NewNullInt64(a1.ID),
				SrcFilename:     "/source/file1.test",
				DestFilename:    "/dest/file1.test",
				Progress:        1,
				TaskNumber:      2,
				Start:           time.Date(2021, 1, 1, 1, 0, 0, 123000, time.Local),
				Step:            types.StepPreTasks,
				Status:          types.StatusPlanned,
			}
			So(db.Insert(t1).Run(), ShouldBeNil)

			t2 := &model.Transfer{
				RuleID:          r2.ID,
				ClientID:        utils.NewNullInt64(cli2.ID),
				RemoteAccountID: utils.NewNullInt64(a2.ID),
				SrcFilename:     "/source/file2.test",
				DestFilename:    "/dest/file2.test",
				Start:           time.Date(2021, 1, 1, 2, 0, 0, 234000, time.Local),
				Step:            types.StepPostTasks,
				Status:          types.StatusError,
			}
			So(db.Insert(t2).Run(), ShouldBeNil)

			t3 := &model.Transfer{
				RuleID:          r2.ID,
				ClientID:        utils.NewNullInt64(cli2.ID),
				RemoteAccountID: utils.NewNullInt64(a1.ID),
				SrcFilename:     "/source/file3.test",
				DestFilename:    "/dest/file3.test",
				Start:           time.Date(2021, 1, 1, 3, 0, 0, 345000, time.Local),
				Step:            types.StepData,
				Status:          types.StatusPaused,
			}
			So(db.Insert(t3).Run(), ShouldBeNil)

			t1.Step = types.StepFinalization
			So(db.Update(t1).Run(), ShouldBeNil)

			trans1 := fromTransfer(db, t1)
			trans2 := fromTransfer(db, t2)
			trans3 := fromTransfer(db, t3)

			// add a transfer from another gateway
			owner := conf.GlobalConfig.GatewayName
			conf.GlobalConfig.GatewayName = "foobar"
			other := &model.Transfer{
				RuleID:          r1.ID,
				ClientID:        utils.NewNullInt64(cli1.ID),
				RemoteAccountID: utils.NewNullInt64(a1.ID),
				SrcFilename:     "/source/file4.test",
				DestFilename:    "/dest/file4.test",
				Start:           time.Date(2021, 1, 1, 1, 0, 0, 0, time.Local),
			}
			So(db.Insert(other).Run(), ShouldBeNil)

			conf.GlobalConfig.GatewayName = owner

			Convey("Given a request with no parameters", func() {
				req, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})
					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then it should return 2 transfer", func() {
						expected["transfers"] = []OutTransfer{*trans1, *trans2, *trans3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a valid 'rule' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, "?rule="+r2.Name, nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})
					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then it should return 2 transfer", func() {
						expected["transfers"] = []OutTransfer{*trans2, *trans3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a valid 'status' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, "?status=PLANNED", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})
					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then it should return all transfer", func() {
						expected["transfers"] = []OutTransfer{*trans1}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a valid 'start' parameter", func() {
				date := time.Date(2021, 1, 1, 2, 30, 0, 0, time.Local).Format(time.RFC3339)
				req, err := http.NewRequest(http.MethodGet,
					fmt.Sprintf("?start=%s", url.QueryEscape(date)), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})
					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then it should return all transfer", func() {
						expected["transfers"] = []OutTransfer{*trans3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})
		})
	})
}

func TestResumeTransfer(t *testing.T) {
	Convey("Testing the transfer resume handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_transfer_resume_test")
		db := database.TestDatabase(c)
		handler := resumeTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 transfer in error", func() {
			client := &model.Client{Name: "test_client", Protocol: testProto1}
			So(db.Insert(client).Run(), ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:     "test_server",
				Protocol: client.Protocol,
				Address:  "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "toto",
				Password:      "titi",
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			rule := &model.Rule{Name: "test_rule", IsSend: false, Path: "path"}
			So(db.Insert(rule).Run(), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:          rule.ID,
				ClientID:        utils.NewNullInt64(client.ID),
				RemoteAccountID: utils.NewNullInt64(account.ID),
				SrcFilename:     "file.src",
				DestFilename:    "file.dst",
				Start:           time.Date(2020, 1, 1, 1, 0, 0, 0, time.Local),
				Status:          types.StatusError,
				Step:            types.StepData,
				ErrCode:         types.TeDataTransfer,
				ErrDetails:      "transfer failed",
				Progress:        10,
				TaskNumber:      0,
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			Convey("Given a request with the valid transfer ID parameter", func() {
				id := utils.FormatInt(trans.ID)
				req, err := http.NewRequest(http.MethodPut, "", nil)
				So(err, ShouldBeNil)

				req = mux.SetURLVars(req, map[string]string{"transfer": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeBlank)
					})

					Convey("Then it should reply 'Accepted'", func() {
						So(w.Code, ShouldEqual, http.StatusAccepted)
					})

					Convey("Then the transfer should have been reprogrammed", func() {
						var transfers model.Transfers

						So(db.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldNotBeEmpty)
						So(transfers[0], ShouldResemble, &model.Transfer{
							ID:               trans.ID,
							Owner:            conf.GlobalConfig.GatewayName,
							RemoteTransferID: trans.RemoteTransferID,
							RuleID:           rule.ID,
							ClientID:         utils.NewNullInt64(client.ID),
							RemoteAccountID:  utils.NewNullInt64(account.ID),
							SrcFilename:      "file.src",
							DestFilename:     "file.dst",
							Start:            trans.Start.Local(),
							Status:           types.StatusPlanned,
							Step:             types.StepData,
							Progress:         10,
							TaskNumber:       0,
						})
					})
				})
			})
		})
	})
}

func TestPauseTransfer(t *testing.T) {
	Convey("Testing the transfer pause handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_transfer_pause_test")
		db := database.TestDatabase(c)
		handler := pauseTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 planned transfer", func() {
			client := &model.Client{Name: "test_client", Protocol: testProto1}
			So(db.Insert(client).Run(), ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:     "test_server",
				Protocol: client.Protocol,
				Address:  "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "toto",
				Password:      "titi",
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			rule := &model.Rule{Name: "test_rule", IsSend: false, Path: "path"}
			So(db.Insert(rule).Run(), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:          rule.ID,
				ClientID:        utils.NewNullInt64(client.ID),
				RemoteAccountID: utils.NewNullInt64(account.ID),
				SrcFilename:     "file.src",
				DestFilename:    "file.dst",
				Start:           time.Date(2020, 1, 2, 3, 4, 5, 678000, time.Local),
				Status:          types.StatusPlanned,
				Step:            types.StepData,
				Progress:        10,
				TaskNumber:      0,
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			Convey("Given a request with the valid transfer ID parameter", func() {
				id := utils.FormatInt(trans.ID)
				req, err := http.NewRequest(http.MethodPut, "", nil)
				So(err, ShouldBeNil)

				req = mux.SetURLVars(req, map[string]string{"transfer": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeBlank)
					})

					Convey("Then it should reply 'Accepted'", func() {
						So(w.Code, ShouldEqual, http.StatusAccepted)
					})

					Convey("Then the transfer should have been paused", func() {
						var transfers model.Transfers

						So(db.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldNotBeEmpty)
						So(transfers[0], ShouldResemble, &model.Transfer{
							ID:               trans.ID,
							RemoteTransferID: trans.RemoteTransferID,
							Owner:            conf.GlobalConfig.GatewayName,
							RuleID:           rule.ID,
							ClientID:         utils.NewNullInt64(client.ID),
							RemoteAccountID:  utils.NewNullInt64(account.ID),
							SrcFilename:      "file.src",
							DestFilename:     "file.dst",
							Start:            trans.Start.Local(),
							Status:           types.StatusPaused,
							Step:             types.StepData,
							Progress:         10,
							TaskNumber:       0,
						})
					})
				})
			})
		})
	})
}

func TestCancelTransfer(t *testing.T) {
	Convey("Testing the transfer resume handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_transfer_cancel_test")
		db := database.TestDatabase(c)
		handler := cancelTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 planned transfer", func() {
			client := &model.Client{Name: "test_client", Protocol: testProto1}
			So(db.Insert(client).Run(), ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:     "test_server",
				Protocol: client.Protocol,
				Address:  "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "toto",
				Password:      "titi",
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			rule := &model.Rule{Name: "test_rule", IsSend: false, Path: "path"}
			So(db.Insert(rule).Run(), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:          rule.ID,
				ClientID:        utils.NewNullInt64(client.ID),
				RemoteAccountID: utils.NewNullInt64(account.ID),
				SrcFilename:     "file.src",
				DestFilename:    "file.dst",
				Start:           time.Date(2030, 1, 1, 1, 0, 0, 0, time.Local),
				Status:          types.StatusError,
				Step:            types.StepNone,
				ErrCode:         types.TeUnknown,
				ErrDetails:      "this is an error",
				Progress:        0,
				TaskNumber:      0,
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			Convey("Given a request with the valid transfer ID parameter", func() {
				id := utils.FormatInt(trans.ID)
				req, err := http.NewRequest(http.MethodPut, "", nil)
				So(err, ShouldBeNil)

				req = mux.SetURLVars(req, map[string]string{"transfer": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeBlank)
					})

					Convey("Then it should reply 'Accepted'", func() {
						So(w.Code, ShouldEqual, http.StatusAccepted)
					})

					Convey("Then the transfer should have been canceled", func() {
						var hist model.HistoryEntries

						So(db.Select(&hist).Run(), ShouldBeNil)
						So(hist, ShouldNotBeEmpty)
						So(hist[0], ShouldResemble, &model.HistoryEntry{
							ID:               trans.ID,
							Owner:            conf.GlobalConfig.GatewayName,
							RemoteTransferID: trans.RemoteTransferID,
							IsServer:         trans.IsServer(),
							IsSend:           rule.IsSend,
							Client:           client.Name,
							Account:          account.Login,
							Agent:            partner.Name,
							Protocol:         testProto1,
							SrcFilename:      "file.src",
							DestFilename:     "file.dst",
							Rule:             rule.Name,
							Start:            trans.Start,
							Stop:             time.Time{},
							Status:           types.StatusCancelled,
							ErrCode:          trans.ErrCode,
							ErrDetails:       trans.ErrDetails,
							Step:             trans.Step,
							Progress:         trans.Progress,
							TaskNumber:       trans.TaskNumber,
						})
					})
				})
			})
		})
	})
}

func TestRestartTransfer(t *testing.T) {
	Convey("Testing the transfer restart handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_history_restart_test")
		db := database.TestDatabase(c)
		handler := retryTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 transfer history", func() {
			client := &model.Client{Name: "client", Protocol: testProto1}
			So(db.Insert(client).Run(), ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:     "partner",
				Protocol: client.Protocol,
				Address:  "localhost:2022",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "toto",
				Password:      "titi",
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			rule := model.Rule{Name: "rule", IsSend: true, Path: "path"}
			So(db.Insert(&rule).Run(), ShouldBeNil)

			h := &model.HistoryEntry{
				ID:               2,
				RemoteTransferID: "1234",
				IsServer:         false,
				IsSend:           rule.IsSend,
				Rule:             rule.Name,
				Client:           client.Name,
				Account:          account.Login,
				Agent:            partner.Name,
				Protocol:         testProto1,
				SrcFilename:      "/source/file.test",
				DestFilename:     "/dest/file.test",
				LocalPath:        *mkURL("file:/local/file.test"),
				RemotePath:       "/remote/file.test",
				Start:            time.Date(2019, 1, 1, 0, 0, 0, 0, time.Local),
				Stop:             time.Date(2019, 1, 1, 1, 0, 0, 0, time.Local),
				Status:           types.StatusCancelled,
			}
			So(db.Insert(h).Run(), ShouldBeNil)

			id := utils.FormatInt(h.ID)

			Convey("Given a request with the valid transfer history ID parameter", func() {
				dateStr := url.QueryEscape(h.Start.Format(time.RFC3339Nano))

				uri := fmt.Sprintf("%s/%s/restart?date=%s", historyURI, id, dateStr)
				req, err := http.NewRequest(http.MethodPut, uri, nil)
				So(err, ShouldBeNil)

				req = mux.SetURLVars(req, map[string]string{"transfer": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)
					res := w.Result() //nolint:bodyclose // body is closed the line after !?

					defer res.Body.Close()

					Convey("Then it should reply 'CREATED'", func() {
						So(res.StatusCode, ShouldEqual, http.StatusCreated)
					})

					Convey("Then the response body should be empty", func() {
						body, err := io.ReadAll(res.Body)
						So(err, ShouldBeNil)
						So(string(body), ShouldBeBlank)
					})

					Convey("Then the 'Location' header should contain the URI "+
						"of the new transfer", func() {
						loc, err := res.Location()
						So(err, ShouldBeNil)
						So(loc.String(), ShouldStartWith, transferURI)
					})

					Convey("Then the transfer should have been reprogrammed", func() {
						var transfers model.Transfers

						So(db.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldNotBeEmpty)

						So(transfers[0].ID, ShouldEqual, 1)
						So(transfers[0].RemoteTransferID, ShouldNotEqual, h.RemoteTransferID)
						So(transfers[0].RuleID, ShouldEqual, rule.ID)
						So(transfers[0].RemoteAccountID.Int64, ShouldEqual, account.ID)
						So(transfers[0].SrcFilename, ShouldEqual, h.SrcFilename)
						So(transfers[0].DestFilename, ShouldEqual, h.DestFilename)
						So(transfers[0].Start, ShouldEqual, h.Start)
						So(transfers[0].Status, ShouldEqual, types.StatusPlanned)
						So(transfers[0].Owner, ShouldEqual, h.Owner)
					})
				})
			})

			Convey("Given a request with a non-existing transfer history ID parameter", func() {
				uri := path.Join(historyURI, "1000")
				r, err := http.NewRequest(http.MethodGet, uri, nil)
				So(err, ShouldBeNil)

				r = mux.SetURLVars(r, map[string]string{"history": "1000"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}

func TestCancelTransfers(t *testing.T) {
	Convey("Testing the transfers multi cancel handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_transfers_cancel_test")
		db := database.TestDatabase(c)
		handler := cancelTransfers(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 planned & 1 error transfer", func() {
			client := &model.Client{Name: "client", Protocol: testProto1}
			So(db.Insert(client).Run(), ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:     "test_server",
				Protocol: client.Protocol,
				Address:  "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "toto",
				Password:      "titi",
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			rule := &model.Rule{Name: "test_rule", IsSend: false, Path: "path"}
			So(db.Insert(rule).Run(), ShouldBeNil)

			transPlan := &model.Transfer{
				RuleID:          rule.ID,
				ClientID:        utils.NewNullInt64(client.ID),
				RemoteAccountID: utils.NewNullInt64(account.ID),
				SrcFilename:     "file.src",
				DestFilename:    "file.dst",
				Start:           time.Date(2030, 1, 1, 1, 0, 0, 0, time.Local),
				Status:          types.StatusPlanned,
				Step:            types.StepNone,
				Progress:        0,
				TaskNumber:      0,
			}
			So(db.Insert(transPlan).Run(), ShouldBeNil)

			transErr := &model.Transfer{
				RuleID:          rule.ID,
				ClientID:        utils.NewNullInt64(client.ID),
				RemoteAccountID: utils.NewNullInt64(account.ID),
				SrcFilename:     "file.src",
				DestFilename:    "file.dst",
				Start:           time.Date(2030, 1, 1, 1, 0, 0, 0, time.Local),
				Status:          types.StatusError,
				Step:            types.StepData,
				ErrCode:         types.TeDataTransfer,
				ErrDetails:      "error msg",
				Progress:        0,
				TaskNumber:      0,
			}
			So(db.Insert(transErr).Run(), ShouldBeNil)

			Convey("Given a request with the 'planned' target", func() {
				req, err := http.NewRequest(http.MethodDelete, "?target=planned", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldEqual, "Transfers canceled successfully")
					})

					Convey("Then it should reply 'Accepted'", func() {
						So(w.Code, ShouldEqual, http.StatusAccepted)
					})

					Convey("Then the planned transfer should have been canceled", func() {
						exp := &model.HistoryEntry{
							ID:               transPlan.ID,
							Owner:            conf.GlobalConfig.GatewayName,
							RemoteTransferID: transPlan.RemoteTransferID,
							IsServer:         false,
							IsSend:           false,
							Client:           client.Name,
							Account:          account.Login,
							Agent:            partner.Name,
							Protocol:         testProto1,
							SrcFilename:      "file.src",
							DestFilename:     "file.dst",
							Rule:             rule.Name,
							Start:            time.Date(2030, 1, 1, 1, 0, 0, 0, time.Local),
							Stop:             time.Time{},
							Status:           types.StatusCancelled,
							Step:             types.StepNone,
							Progress:         0,
							TaskNumber:       0,
						}

						var hist model.HistoryEntries

						So(db.Select(&hist).Run(), ShouldBeNil)
						So(hist, ShouldNotBeEmpty)
						So(hist[0], ShouldResemble, exp)
					})

					Convey("Then the other non-planned transfers should be unaffected", func() {
						var trans model.Transfers

						So(db.Select(&trans).Run(), ShouldBeNil)
						So(trans, ShouldHaveLength, 1)
						So(trans[0], ShouldResemble, transErr)
					})
				})
			})

			Convey("Given a request with the 'running' target", func() {
				req, err := http.NewRequest(http.MethodDelete, "?target=running", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldEqual, "Transfers canceled successfully")
					})

					Convey("Then it should reply 'Accepted'", func() {
						So(w.Code, ShouldEqual, http.StatusAccepted)
					})

					Convey("Then the other non-running transfers should be unaffected", func() {
						var trans model.Transfers

						So(db.Select(&trans).Run(), ShouldBeNil)
						So(trans, ShouldHaveLength, 2)
						So(trans[0], ShouldResemble, transPlan)
						So(trans[1], ShouldResemble, transErr)
					})
				})
			})
		})
	})
}
