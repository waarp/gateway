package rest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strconv"
	"testing"
	"time"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

const historyURI = "http://localhost:8080/api/history"

func TestGetHistory(t *testing.T) {
	logger := log.NewLogger("rest_history_get_test")

	Convey("Testing the transfer history get handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := getHistory(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 transfer history", func() {
			h := &model.TransferHistory{
				ID:             1,
				IsServer:       true,
				IsSend:         false,
				Rule:           "rule",
				Account:        "acc",
				Agent:          "server",
				Protocol:       "sftp",
				SourceFilename: "file.test",
				DestFilename:   "file.test",
				Start:          time.Date(2021, 1, 2, 3, 4, 5, 678000, time.Local),
				Stop:           time.Date(2021, 2, 3, 4, 5, 6, 789000, time.Local),
				Status:         "DONE",
			}
			So(db.Insert(h).Run(), ShouldBeNil)

			id := strconv.FormatUint(h.ID, 10)

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

						exp, err := json.Marshal(FromHistory(h))

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
		})
	})
}

func TestListHistory(t *testing.T) {
	logger := log.NewLogger("rest_history_get_test")

	Convey("Testing the transfer history list handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := listHistory(logger, db)
		w := httptest.NewRecorder()

		expected := map[string][]OutHistory{}

		Convey("Given a database with 4 transfer history", func() {
			h1 := &model.TransferHistory{
				ID:             1,
				IsServer:       true,
				IsSend:         false,
				Account:        "from1",
				Agent:          "to3",
				Protocol:       "sftp",
				Rule:           "rule1",
				Start:          time.Date(2019, 1, 1, 1, 0, 0, 1000, time.Local),
				Stop:           time.Date(2019, 1, 1, 3, 0, 0, 1000, time.Local),
				Status:         types.StatusDone,
				SourceFilename: "file.test",
				DestFilename:   "file.test",
			}
			So(db.Insert(h1).Run(), ShouldBeNil)

			h2 := &model.TransferHistory{
				ID:             2,
				IsServer:       false,
				IsSend:         false,
				Account:        "from2",
				Agent:          "to1",
				Protocol:       "sftp",
				Rule:           "rule2",
				Start:          time.Date(2019, 1, 1, 2, 0, 0, 2000, time.Local),
				Stop:           time.Date(2019, 1, 1, 4, 0, 0, 2000, time.Local),
				Status:         types.StatusCancelled,
				SourceFilename: "file.test",
				DestFilename:   "file.test",
			}
			So(db.Insert(h2).Run(), ShouldBeNil)

			h3 := &model.TransferHistory{
				ID:             3,
				IsServer:       false,
				IsSend:         true,
				Account:        "from3",
				Agent:          "to2",
				Protocol:       "sftp",
				Rule:           "rule1",
				Start:          time.Date(2019, 1, 1, 3, 0, 0, 3000, time.Local),
				Stop:           time.Date(2019, 1, 1, 5, 0, 0, 3000, time.Local),
				Status:         types.StatusCancelled,
				SourceFilename: "file.test",
				DestFilename:   "file.test",
			}
			So(db.Insert(h3).Run(), ShouldBeNil)

			h4 := &model.TransferHistory{
				ID:             4,
				IsServer:       false,
				IsSend:         true,
				Account:        "from4",
				Agent:          "to3",
				Protocol:       "sftp",
				Rule:           "rule2",
				Start:          time.Date(2019, 1, 1, 4, 0, 0, 4000, time.Local),
				Stop:           time.Date(2019, 1, 1, 6, 0, 0, 4000, time.Local),
				Status:         types.StatusDone,
				SourceFilename: "file.test",
				DestFilename:   "file.test",
			}
			So(db.Insert(h4).Run(), ShouldBeNil)

			hist1 := *FromHistory(h1)
			hist2 := *FromHistory(h2)
			hist3 := *FromHistory(h3)
			hist4 := *FromHistory(h4)

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
						expected["history"] = []OutHistory{hist1, hist2}
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
						expected["history"] = []OutHistory{hist2, hist3}
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
						expected["history"] = []OutHistory{hist1, hist3}
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
						So(w.Body.String(), ShouldEqual, "'toto' is not a valid protocol\n")
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
						expected["history"] = []OutHistory{hist1, hist2, hist3, hist4}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 1 valid 'start' parameter", func() {
				date := h3.Start.Format(time.RFC3339Nano)
				req, err := http.NewRequest(http.MethodGet,
					fmt.Sprintf("?start=%s", url.QueryEscape(date)), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 3 transfer history", func() {
						expected["history"] = []OutHistory{hist3, hist4}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 1 valid 'stop' parameter", func() {
				date := h2.Stop.Format(time.RFC3339Nano)
				req, err := http.NewRequest(http.MethodGet,
					fmt.Sprintf("?stop=%s", url.QueryEscape(date)), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 1 transfer history", func() {
						expected["history"] = []OutHistory{hist1, hist2}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 1 valid 'stop' and 1 valid 'start' parameter", func() {
				start := h2.Start.Add(-time.Minute)
				stop := h3.Stop.Add(time.Minute)
				req, err := http.NewRequest(http.MethodGet,
					fmt.Sprintf("?start=%s&stop=%s", url.QueryEscape(start.Format(time.RFC3339Nano)),
						url.QueryEscape(stop.Format(time.RFC3339Nano))), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 1 transfer history", func() {
						expected["history"] = []OutHistory{hist2, hist3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 1 valid 'status' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("?status=%s", "DONE"), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 1 transfer history", func() {
						expected["history"] = []OutHistory{hist1, hist4}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})
		})
	})
}

func TestRestartTransfer(t *testing.T) {
	logger := log.NewLogger("rest_history_restart_test")

	Convey("Testing the transfer restart handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := retryTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 transfer history", func() {
			partner := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
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

			h := &model.TransferHistory{
				ID:             2,
				IsServer:       false,
				IsSend:         rule.IsSend,
				Rule:           rule.Name,
				Account:        account.Login,
				Agent:          partner.Name,
				Protocol:       "test",
				SourceFilename: "file.test",
				DestFilename:   "file.test",
				Start:          time.Date(2019, 1, 1, 0, 0, 0, 0, time.Local),
				Stop:           time.Date(2019, 1, 1, 1, 0, 0, 0, time.Local),
				Status:         types.StatusCancelled,
			}
			So(db.Insert(h).Run(), ShouldBeNil)

			id := strconv.FormatUint(h.ID, 10)

			Convey("Given a request with the valid transfer history ID parameter", func() {
				dateStr := url.QueryEscape(h.Start.Format(time.RFC3339Nano))

				uri := fmt.Sprintf("%s/%s/restart?date=%s", historyURI, id, dateStr)
				req, err := http.NewRequest(http.MethodPut, uri, nil)
				So(err, ShouldBeNil)
				req = mux.SetURLVars(req, map[string]string{"history": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)
					res := w.Result()

					Convey("Then it should reply 'CREATED'", func() {
						So(res.StatusCode, ShouldEqual, http.StatusCreated)
					})

					Convey("Then the response body should be empty", func() {
						body, err := ioutil.ReadAll(res.Body)
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
						expected := model.Transfer{
							ID:         1,
							RuleID:     rule.ID,
							IsServer:   false,
							AgentID:    partner.ID,
							AccountID:  account.ID,
							SourceFile: h.SourceFilename,
							DestFile:   h.DestFilename,
							Start:      h.Start,
							Status:     types.StatusPlanned,
							Owner:      h.Owner,
						}

						var transfers model.Transfers
						So(db.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldNotBeEmpty)
						So(transfers[0], ShouldResemble, expected)
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
