package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
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

const historyURI = "http://localhost:8080/api/history"

func TestGetHistory(t *testing.T) {
	Convey("Testing the transfer history get handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_history_get_test")
		db := database.TestDatabase(c)
		handler := getHistory(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 transfer history", func() {
			h := &model.HistoryEntry{
				ID:               1,
				RemoteTransferID: "1234",
				IsServer:         true,
				IsSend:           false,
				Rule:             "rule",
				Account:          "acc",
				Agent:            "server",
				Protocol:         "sftp",
				SrcFilename:      "file.test",
				DestFilename:     "file.test",
				LocalPath:        *mkURL("file:/local/file.test"),
				RemotePath:       "/remote/file.test",
				Start:            time.Date(2021, 1, 2, 3, 4, 5, 678000, time.Local),
				Stop:             time.Date(2021, 2, 3, 4, 5, 6, 789000, time.Local),
				Status:           "DONE",
			}
			So(db.Insert(h).Run(), ShouldBeNil)

			infos := map[string]any{"key1": "val1", "key2": 2}
			So(h.SetTransferInfo(db, infos), ShouldBeNil)

			id := utils.FormatInt(h.ID)

			Convey("Given a request with the valid transfer history ID parameter", func() {
				uri := path.Join(historyURI, id)
				req, err := http.NewRequest(http.MethodGet, uri, nil)
				So(err, ShouldBeNil)

				req = mux.SetURLVars(req, map[string]string{"history": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain "+
						"'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested transfer history "+
						"in JSON format", func() {
						jHist, err := FromHistory(db, h)
						So(err, ShouldBeNil)
						exp, err := json.Marshal(jHist)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
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

			Convey("Given a database with an unown 1 transfer history ", func() {
				gwname := conf.GlobalConfig.GatewayName
				conf.GlobalConfig.GatewayName = "other"

				h2 := &model.HistoryEntry{
					ID:               2,
					RemoteTransferID: "1235",
					IsServer:         true,
					IsSend:           false,
					Rule:             "rule",
					Account:          "acc",
					Agent:            "server",
					Protocol:         "sftp",
					SrcFilename:      "/source/file.test",
					DestFilename:     "/dest/file.test",
					LocalPath:        *mkURL("file:/local/file.test"),
					RemotePath:       "/remote/file.test",
					Start:            time.Date(2021, 1, 2, 3, 4, 5, 678000, time.Local),
					Stop:             time.Date(2021, 2, 3, 4, 5, 6, 789000, time.Local),
					Status:           "DONE",
				}
				So(db.Insert(h2).Run(), ShouldBeNil)

				infos := map[string]any{"key1": "val1", "key2": 2}
				So(h.SetTransferInfo(db, infos), ShouldBeNil)

				conf.GlobalConfig.GatewayName = gwname

				id2 := utils.FormatInt(h2.ID)

				Convey("Given a request with the unown transfer history ID parameter", func() {
					uri := path.Join(historyURI, id2)
					req, err := http.NewRequest(http.MethodGet, uri, nil)
					So(err, ShouldBeNil)

					req = mux.SetURLVars(req, map[string]string{"history": id2})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, req)

						Convey("Then it should reply with a 'Not Found' error", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})
					})
				})
			})
		})
	})
}

//nolint:maintidx //FIXME factorize the function if possible to improve maintainability
func TestListHistory(t *testing.T) {
	Convey("Testing the transfer history list handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_history_get_test")
		db := database.TestDatabase(c)
		handler := listHistory(logger, db)
		w := httptest.NewRecorder()

		expected := map[string][]OutHistory{}

		Convey("Given a database with 4 transfer history", func() {
			h1 := &model.HistoryEntry{
				ID:               1,
				RemoteTransferID: "1111",
				IsServer:         true,
				IsSend:           false,
				Account:          "from1",
				Agent:            "to3",
				Protocol:         "sftp",
				Rule:             "rule1",
				Start:            time.Date(2019, 1, 1, 1, 0, 0, 1000, time.Local),
				Stop:             time.Date(2019, 1, 1, 3, 0, 0, 1000, time.Local),
				Status:           types.StatusDone,
				SrcFilename:      "/source/file1.test",
				DestFilename:     "/dest/file1.test",
				LocalPath:        *mkURL("file:/local/file1.test"),
				RemotePath:       "/remote/file1.test",
			}
			So(db.Insert(h1).Run(), ShouldBeNil)

			h2 := &model.HistoryEntry{
				ID:               2,
				RemoteTransferID: "2222",
				IsServer:         false,
				IsSend:           false,
				Client:           "with1",
				Account:          "from2",
				Agent:            "to1",
				Protocol:         "sftp",
				Rule:             "rule2",
				Start:            time.Date(2019, 1, 1, 2, 0, 0, 2000, time.Local),
				Stop:             time.Date(2019, 1, 1, 4, 0, 0, 2000, time.Local),
				Status:           types.StatusCancelled,
				SrcFilename:      "/source/file2.test",
				DestFilename:     "/dest/file2.test",
				LocalPath:        *mkURL("file:/local/file2.test"),
				RemotePath:       "/remote/file2.test",
			}
			So(db.Insert(h2).Run(), ShouldBeNil)

			h3 := &model.HistoryEntry{
				ID:               3,
				RemoteTransferID: "3333",
				IsServer:         false,
				IsSend:           true,
				Client:           "with2",
				Account:          "from3",
				Agent:            "to2",
				Protocol:         "sftp",
				Rule:             "rule1",
				Start:            time.Date(2019, 1, 1, 3, 0, 0, 3000, time.Local),
				Stop:             time.Date(2019, 1, 1, 5, 0, 0, 3000, time.Local),
				Status:           types.StatusCancelled,
				SrcFilename:      "/source/file3.test",
				DestFilename:     "/dest/file3.test",
				LocalPath:        *mkURL("/local/file3.test"),
				RemotePath:       "/remote/file3.test",
			}
			So(db.Insert(h3).Run(), ShouldBeNil)

			h4 := &model.HistoryEntry{
				ID:               4,
				RemoteTransferID: "4444",
				IsServer:         false,
				IsSend:           true,
				Client:           "with3",
				Account:          "from4",
				Agent:            "to3",
				Protocol:         "sftp",
				Rule:             "rule2",
				Start:            time.Date(2019, 1, 1, 4, 0, 0, 4000, time.Local),
				Stop:             time.Date(2019, 1, 1, 6, 0, 0, 4000, time.Local),
				Status:           types.StatusDone,
				SrcFilename:      "/source/file4.test",
				DestFilename:     "/dest/file4.test",
				LocalPath:        *mkURL("/local/file4.test"),
				RemotePath:       "/remote/file4.test",
			}
			So(db.Insert(h4).Run(), ShouldBeNil)

			hist1, err := FromHistory(db, h1)
			So(err, ShouldBeNil)
			hist2, err := FromHistory(db, h2)
			So(err, ShouldBeNil)
			hist3, err := FromHistory(db, h3)
			So(err, ShouldBeNil)
			hist4, err := FromHistory(db, h4)
			So(err, ShouldBeNil)

			Convey("Given a request with 2 valid 'requester' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, "?requester=from1&requester=from2", nil)
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

					Convey("Then it should return 2 transfer history", func() {
						expected["history"] = []OutHistory{*hist1, *hist2}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 2 valid 'requested' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, "?requested=to1&requested=to2", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 2 transfer history", func() {
						expected["history"] = []OutHistory{*hist2, *hist3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 1 valid 'rule' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, "?rule=rule1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 2 transfer history", func() {
						expected["history"] = []OutHistory{*hist1, *hist3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 1 valid and 1 invalid 'protocol' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, "?protocol=sftp&protocol=toto", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply with a 'Bad Request' error", func() {
						So(w.Code, ShouldEqual, http.StatusBadRequest)
					})

					Convey("Then the response body should contain "+
						"a message stating the protocol is unknown", func() {
						So(w.Body.String(), ShouldEqual, "\"toto\" is not a valid protocol\n")
					})
				})
			})

			Convey("Given a request with 1 valid 'protocol' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, "?protocol=sftp", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return all 4 transfer history", func() {
						expected["history"] = []OutHistory{*hist1, *hist2, *hist3, *hist4}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 1 valid 'start' parameter", func() {
				date := time.Date(2019, 1, 1, 2, 30, 0, 0, time.Local).Format(time.RFC3339)
				req, err := http.NewRequest(http.MethodGet,
					fmt.Sprintf("?start=%s", url.QueryEscape(date)), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 3 transfer history", func() {
						expected["history"] = []OutHistory{*hist3, *hist4}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 1 valid 'stop' parameter", func() {
				date := time.Date(2019, 1, 1, 4, 30, 0, 0, time.Local).Format(time.RFC3339)
				req, err := http.NewRequest(http.MethodGet,
					fmt.Sprintf("?stop=%s", url.QueryEscape(date)), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 1 transfer history", func() {
						expected["history"] = []OutHistory{*hist1, *hist2}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 1 valid 'stop' and 1 valid 'start' parameter", func() {
				start := time.Date(2019, 1, 1, 1, 30, 0, 0, time.Local).Format(time.RFC3339)
				stop := time.Date(2019, 1, 1, 5, 30, 0, 0, time.Local).Format(time.RFC3339)
				req, err := http.NewRequest(http.MethodGet,
					fmt.Sprintf("?start=%s&stop=%s", url.QueryEscape(start),
						url.QueryEscape(stop)), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 1 transfer history", func() {
						expected["history"] = []OutHistory{*hist2, *hist3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 1 valid 'status' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("?status=%s",
					types.StatusDone), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 1 transfer history", func() {
						expected["history"] = []OutHistory{*hist1, *hist4}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})
		})
	})
}

func TestRestartHistory(t *testing.T) {
	Convey("Testing the transfer restart handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_history_restart_test")
		db := database.TestDatabase(c)
		handler := retryHistory(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 transfer history", func() {
			client := &model.Client{Name: "test_client", Protocol: testProto1}
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
				Protocol:         client.Protocol,
				SrcFilename:      "/source/file1.test",
				DestFilename:     "/dest/file1.test",
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

				req = mux.SetURLVars(req, map[string]string{"history": id})

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
