package rest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

const historyURI = "http://localhost:8080" + APIPath + HistoryPath + "/"

func TestGetHistory(t *testing.T) {
	logger := log.NewLogger("rest_history_get_test")

	Convey("Testing the transfer history get handler", t, func() {
		db := database.GetTestDatabase()
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
				Start:          time.Date(2019, 01, 01, 00, 00, 00, 00, time.UTC),
				Stop:           time.Date(2019, 01, 01, 01, 00, 00, 00, time.UTC),
				Status:         "DONE",
			}
			So(db.Create(h), ShouldBeNil)

			id := strconv.FormatUint(h.ID, 10)
			h.Start = h.Start.Local()
			h.Stop = h.Stop.Local()

			Convey("Given a request with the valid transfer history ID parameter", func() {
				req, err := http.NewRequest(http.MethodGet, historyURI+id, nil)
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
				r, err := http.NewRequest(http.MethodGet, historyURI+"1000", nil)
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

	Convey("Testing the transfer history list handler", t, func() {
		db := database.GetTestDatabase()
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
				Start:          time.Date(2019, 01, 01, 02, 00, 00, 00, time.UTC),
				Stop:           time.Date(2019, 01, 01, 06, 00, 00, 00, time.UTC),
				Status:         "DONE",
				SourceFilename: "file.test",
				DestFilename:   "file.test",
			}
			So(db.Create(h1), ShouldBeNil)
			h1.Start = h1.Start.Local()
			h1.Stop = h1.Stop.Local()

			h2 := &model.TransferHistory{
				ID:             2,
				IsServer:       false,
				IsSend:         false,
				Account:        "from2",
				Agent:          "to1",
				Protocol:       "sftp",
				Rule:           "rule2",
				Start:          time.Date(2019, 01, 01, 01, 00, 00, 00, time.UTC),
				Stop:           time.Date(2019, 01, 01, 07, 00, 00, 00, time.UTC),
				Status:         "ERROR",
				SourceFilename: "file.test",
				DestFilename:   "file.test",
			}
			So(db.Create(h2), ShouldBeNil)
			h2.Start = h2.Start.Local()
			h2.Stop = h2.Stop.Local()

			h3 := &model.TransferHistory{
				ID:             3,
				IsServer:       false,
				IsSend:         true,
				Account:        "from3",
				Agent:          "to2",
				Protocol:       "sftp",
				Rule:           "rule1",
				Start:          time.Date(2019, 01, 01, 03, 00, 00, 00, time.UTC),
				Stop:           time.Date(2019, 01, 01, 8, 00, 00, 00, time.UTC),
				Status:         "ERROR",
				SourceFilename: "file.test",
				DestFilename:   "file.test",
			}
			So(db.Create(h3), ShouldBeNil)
			h3.Start = h3.Start.Local()
			h3.Stop = h3.Stop.Local()

			h4 := &model.TransferHistory{
				ID:             4,
				IsServer:       false,
				IsSend:         true,
				Account:        "from4",
				Agent:          "to3",
				Protocol:       "sftp",
				Rule:           "rule2",
				Start:          time.Date(2019, 01, 01, 04, 00, 00, 00, time.UTC),
				Stop:           time.Date(2019, 01, 01, 05, 00, 00, 00, time.UTC),
				Status:         "DONE",
				SourceFilename: "file.test",
				DestFilename:   "file.test",
			}
			So(db.Create(h4), ShouldBeNil)
			h4.Start = h4.Start.Local()
			h4.Stop = h4.Stop.Local()

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
						expected["history"] = []OutHistory{hist2, hist1}
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
						expected["history"] = []OutHistory{hist2, hist1, hist3, hist4}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 1 valid 'start' parameter", func() {
				date := h1.Start.Add(-time.Minute)
				req, err := http.NewRequest(http.MethodGet,
					fmt.Sprintf("?start=%s", url.QueryEscape(date.Format(time.RFC3339))), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 3 transfer history", func() {
						expected["history"] = []OutHistory{hist1, hist3, hist4}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 1 valid 'stop' parameter", func() {
				date := h4.Stop.Add(time.Minute)
				req, err := http.NewRequest(http.MethodGet,
					fmt.Sprintf("?stop=%s", url.QueryEscape(date.Format(time.RFC3339))), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 1 transfer history", func() {
						expected["history"] = []OutHistory{hist4}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 1 valid 'stop' and 1 valid 'start' parameter", func() {
				start := h3.Start.Add(-time.Minute)
				stop := h3.Stop.Add(time.Minute)
				req, err := http.NewRequest(http.MethodGet,
					fmt.Sprintf("?start=%s&stop=%s", url.QueryEscape(start.Format(time.RFC3339)),
						url.QueryEscape(stop.Format(time.RFC3339))), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 1 transfer history", func() {
						expected["history"] = []OutHistory{hist3, hist4}
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

	Convey("Testing the transfer restart handler", t, func() {
		db := database.GetTestDatabase()
		handler := retryTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 transfer history", func() {
			partner := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(partner), ShouldBeNil)

			cert := &model.Cert{
				OwnerType:   partner.TableName(),
				OwnerID:     partner.ID,
				Name:        "sftp_cert",
				PrivateKey:  nil,
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(cert), ShouldBeNil)

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "toto",
				Password:      []byte("titi"),
			}
			So(db.Create(account), ShouldBeNil)

			rule := model.Rule{Name: "rule", IsSend: true}
			So(db.Create(&rule), ShouldBeNil)

			h := &model.TransferHistory{
				ID:             1,
				IsServer:       false,
				IsSend:         rule.IsSend,
				Rule:           rule.Name,
				Account:        account.Login,
				Agent:          partner.Name,
				Protocol:       "test",
				SourceFilename: "file.test",
				DestFilename:   "file.test",
				Start:          time.Date(2019, 01, 01, 00, 00, 00, 00, time.UTC),
				Stop:           time.Date(2019, 01, 01, 01, 00, 00, 00, time.UTC),
				Status:         model.StatusError,
			}
			So(db.Create(h), ShouldBeNil)

			id := strconv.FormatUint(h.ID, 10)
			h.Start = h.Start.Local()
			h.Stop = h.Stop.Local()

			Convey("Given a request with the valid transfer history ID parameter", func() {
				date := time.Now().Add(time.Hour + time.Minute).Truncate(time.Second)
				dateStr := url.QueryEscape(date.Format(time.RFC3339))

				req, err := http.NewRequest(http.MethodPut, historyURI+id+
					"/restart?date="+dateStr, nil)
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
							Start:      date,
							Status:     model.StatusPlanned,
							Owner:      h.Owner,
						}

						var t []model.Transfer
						So(db.Select(&t, nil), ShouldBeNil)
						So(t, ShouldNotBeEmpty)
						So(t[0], ShouldResemble, expected)
					})
				})
			})

			Convey("Given a request with a non-existing transfer history ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, historyURI+"1000", nil)
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
