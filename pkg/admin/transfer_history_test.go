package admin

import (
	"encoding/json"
	"fmt"
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

const historyURI = APIPath + HistoryPath + "/"

func TestGetHistory(t *testing.T) {
	logger := log.NewLogger("rest_history_get_test")

	Convey("Testing the transfer history get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getHistory(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 transfer history", func() {
			h := model.TransferHistory{
				ID:       1,
				Rule:     "rule",
				Source:   "acc",
				Dest:     "server",
				Protocol: "sftp",
				Filename: "file.test",
				Start:    time.Date(2019, 01, 01, 00, 00, 00, 00, time.UTC),
				Stop:     time.Date(2019, 01, 01, 01, 00, 00, 00, time.UTC),
				Status:   "DONE",
			}
			So(db.Create(&h), ShouldBeNil)

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

						exp, err := json.Marshal(&h)

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

		expected := map[string][]interface{}{}

		Convey("Given a database with 4 transfer history", func() {
			h1 := model.TransferHistory{
				ID:       1,
				Source:   "from1",
				Dest:     "to3",
				Protocol: "sftp",
				Rule:     "rule1",
				Start:    time.Date(2019, 01, 01, 02, 00, 00, 00, time.UTC),
				Stop:     time.Date(2019, 01, 01, 06, 00, 00, 00, time.UTC),
				Status:   "DONE",
				Filename: "file.test",
			}
			So(db.Create(&h1), ShouldBeNil)
			h1.Start = h1.Start.Local()
			h1.Stop = h1.Stop.Local()

			h2 := model.TransferHistory{
				ID:       2,
				Source:   "from2",
				Dest:     "to1",
				Protocol: "sftp",
				Rule:     "rule2",
				Start:    time.Date(2019, 01, 01, 01, 00, 00, 00, time.UTC),
				Stop:     time.Date(2019, 01, 01, 07, 00, 00, 00, time.UTC),
				Status:   "ERROR",
				Filename: "file.test",
			}
			So(db.Create(&h2), ShouldBeNil)
			h2.Start = h2.Start.Local()
			h2.Stop = h2.Stop.Local()

			h3 := model.TransferHistory{
				ID:       3,
				Source:   "from3",
				Dest:     "to2",
				Protocol: "sftp",
				Rule:     "rule1",
				Start:    time.Date(2019, 01, 01, 03, 00, 00, 00, time.UTC),
				Stop:     time.Date(2019, 01, 01, 8, 00, 00, 00, time.UTC),
				Status:   "ERROR",
				Filename: "file.test",
			}
			So(db.Create(&h3), ShouldBeNil)
			h3.Start = h3.Start.Local()
			h3.Stop = h3.Stop.Local()

			h4 := model.TransferHistory{
				ID:       4,
				Source:   "from4",
				Dest:     "to3",
				Protocol: "sftp",
				Rule:     "rule2",
				Start:    time.Date(2019, 01, 01, 04, 00, 00, 00, time.UTC),
				Stop:     time.Date(2019, 01, 01, 05, 00, 00, 00, time.UTC),
				Status:   "DONE",
				Filename: "file.test",
			}
			So(db.Create(&h4), ShouldBeNil)
			h4.Start = h4.Start.Local()
			h4.Stop = h4.Stop.Local()

			Convey("Given a request with 2 valid 'from' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, "?source=from1&source=from2", nil)
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
						expected["history"] = []interface{}{h2, h1}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with 2 valid 'to' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, "?dest=to1&dest=to2", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then it should return 2 transfer history", func() {
						expected["history"] = []interface{}{h2, h3}
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
						expected["history"] = []interface{}{h1, h3}
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
						So(w.Body.String(), ShouldEqual, "toto is not a valid protocol\n")
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
						expected["history"] = []interface{}{h2, h1, h3, h4}
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
						expected["history"] = []interface{}{h1, h3, h4}
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
						expected["history"] = []interface{}{h4}
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
						expected["history"] = []interface{}{h3, h4}
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

					Convey("Then it should return 1 tranfer history", func() {
						expected["history"] = []interface{}{h1, h4}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})
		})
	})
}
