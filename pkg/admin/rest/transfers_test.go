package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

const transferURI = "http://localhost:8080/api/transfers"

func TestAddTransfer(t *testing.T) {
	logger := log.NewLogger("rest_transfer_add_test")

	Convey("Testing the transfer add handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := addTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 partner, 1 certificate & 1 account", func() {
			partner := &model.RemoteAgent{
				Name:        "remote",
				Protocol:    testProto1,
				Address:     "localhost:1",
				ProtoConfig: json.RawMessage(`{}`),
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
					"partner": "remote",
					"account": "toto",
					"isSend": true,
					"file": "test.file"
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
						So(transfers, ShouldNotBeEmpty)

						exp := model.Transfer{
							ID:               1,
							RemoteTransferID: "",
							RuleID:           push.ID,
							IsServer:         false,
							AgentID:          partner.ID,
							AccountID:        account.ID,
							LocalPath:        "test.file",
							RemotePath:       "test.file",
							Filesize:         model.UnknownSize,
							Start:            transfers[0].Start,
							Step:             types.StepNone,
							Status:           types.StatusPlanned,
							Owner:            conf.GlobalConfig.GatewayName,
							Progress:         0,
							TaskNumber:       0,
							Error:            types.TransferError{},
						}

						So(transfers[0], ShouldResemble, exp)
					})
				})
			})

			Convey("Given a new transfer with an invalid rule name", func() {
				body := strings.NewReader(`{
					"rule": "tata",
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
					"partner": "remote",
					"account": "toto",
					"sourcePath": "file.src",
					"destPath": "file.dst"
				}`)

				Convey("When calling the handler", func() {
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
	logger := log.NewLogger("rest_transfer_get_test")

	Convey("Testing the transfer get handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := getTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 transfer", func() {
			partner := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    testProto1,
				Address:     "localhost:1",
				ProtoConfig: json.RawMessage(`{}`),
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "toto",
				Password:      "titi",
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			push := &model.Rule{Name: "push", IsSend: false, Path: "/push"}
			So(db.Insert(push).Run(), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:     push.ID,
				AgentID:    partner.ID,
				AccountID:  account.ID,
				LocalPath:  "/local/file.test",
				RemotePath: "/remote/file.test",
				Start:      time.Date(2021, 1, 1, 1, 0, 0, 0, time.Local),
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			Convey("Given a request with the valid transfer ID parameter", func() {
				id := strconv.FormatUint(trans.ID, 10)
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
						jsonObj, err := FromTransfer(db, trans)
						So(err, ShouldBeNil)
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
	logger := log.NewLogger("rest_transfer_list_test")

	Convey("Testing the transfer list handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := listTransfers(logger, db)
		w := httptest.NewRecorder()

		expected := map[string][]OutTransfer{}

		Convey("Given a database with 2 transfer", func() {
			p1 := &model.RemoteAgent{
				Name:        "part1",
				Protocol:    testProto1,
				Address:     "localhost:1",
				ProtoConfig: json.RawMessage(`{}`),
			}
			So(db.Insert(p1).Run(), ShouldBeNil)

			p2 := &model.RemoteAgent{
				Name:        "part2",
				Protocol:    testProto2,
				Address:     "localhost:2",
				ProtoConfig: json.RawMessage(`{}`),
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
				RuleID:     r1.ID,
				AgentID:    p1.ID,
				AccountID:  a1.ID,
				LocalPath:  "/local/file1.test",
				RemotePath: "/remote/file1.test",
				Progress:   1,
				TaskNumber: 2,
				Start:      time.Date(2021, 1, 1, 1, 0, 0, 123000, time.Local),
				Step:       types.StepPreTasks,
				Status:     types.StatusPlanned,
			}
			So(db.Insert(t1).Run(), ShouldBeNil)

			t2 := &model.Transfer{
				RuleID:     r2.ID,
				AgentID:    p2.ID,
				AccountID:  a2.ID,
				LocalPath:  "/local/file2.test",
				RemotePath: "/remote/file2.test",
				Start:      time.Date(2021, 1, 1, 2, 0, 0, 234000, time.Local),
				Step:       types.StepPostTasks,
				Status:     types.StatusError,
			}
			So(db.Insert(t2).Run(), ShouldBeNil)

			t3 := &model.Transfer{
				RuleID:     r2.ID,
				AgentID:    p1.ID,
				AccountID:  a1.ID,
				LocalPath:  "/local/file3.test",
				RemotePath: "/remote/file3.test",
				Start:      time.Date(2021, 1, 1, 3, 0, 0, 345000, time.Local),
				Step:       types.StepData,
				Status:     types.StatusPaused,
			}
			So(db.Insert(t3).Run(), ShouldBeNil)

			t1.Step = types.StepFinalization
			So(db.Update(t1).Run(), ShouldBeNil)

			trans1, err := FromTransfer(db, t1)
			So(err, ShouldBeNil)
			trans2, err := FromTransfer(db, t2)
			So(err, ShouldBeNil)
			trans3, err := FromTransfer(db, t3)
			So(err, ShouldBeNil)

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
				date := t3.Start
				req, err := http.NewRequest(http.MethodGet,
					fmt.Sprintf("?start=%s", url.QueryEscape(date.Format(time.RFC3339Nano))), nil)
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
	logger := log.NewLogger("rest_transfer_resume_test")

	Convey("Testing the transfer resume handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := resumeTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 transfer in error", func() {
			partner := &model.RemoteAgent{
				Name:        "test_server",
				Protocol:    testProto1,
				Address:     "localhost:1",
				ProtoConfig: json.RawMessage(`{}`),
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
				RuleID:     rule.ID,
				AgentID:    partner.ID,
				AccountID:  account.ID,
				LocalPath:  "file.loc",
				RemotePath: "file.rem",
				Start:      time.Date(2020, 1, 1, 1, 0, 0, 0, time.Local),
				Status:     types.StatusError,
				Step:       types.StepData,
				Error: types.TransferError{
					Code:    types.TeDataTransfer,
					Details: "transfer failed",
				},
				Progress:   10,
				TaskNumber: 0,
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			Convey("Given a request with the valid transfer ID parameter", func() {
				id := strconv.FormatUint(trans.ID, 10)
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
						exp := model.Transfer{
							ID:         trans.ID,
							Owner:      conf.GlobalConfig.GatewayName,
							RuleID:     rule.ID,
							AgentID:    partner.ID,
							AccountID:  account.ID,
							LocalPath:  "file.loc",
							RemotePath: "file.rem",
							Start:      trans.Start.Local(),
							Status:     types.StatusPlanned,
							Step:       types.StepData,
							Error:      types.TransferError{},
							Progress:   10,
							TaskNumber: 0,
						}

						var transfers model.Transfers
						So(db.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldNotBeEmpty)
						So(transfers[0], ShouldResemble, exp)
					})
				})
			})
		})
	})
}

func TestPauseTransfer(t *testing.T) {
	logger := log.NewLogger("rest_transfer_pause_test")

	Convey("Testing the transfer pause handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := pauseTransfer(nil)(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 planned transfer", func() {
			partner := &model.RemoteAgent{
				Name:        "test_server",
				Protocol:    testProto1,
				Address:     "localhost:1",
				ProtoConfig: json.RawMessage(`{}`),
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
				RuleID:     rule.ID,
				AgentID:    partner.ID,
				AccountID:  account.ID,
				LocalPath:  "file.loc",
				RemotePath: "file.rem",
				Start:      time.Date(2020, 1, 2, 3, 4, 5, 678000, time.Local),
				Status:     types.StatusPlanned,
				Step:       types.StepData,
				Error:      types.TransferError{},
				Progress:   10,
				TaskNumber: 0,
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			Convey("Given a request with the valid transfer ID parameter", func() {
				id := strconv.FormatUint(trans.ID, 10)
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
						exp := model.Transfer{
							ID:         trans.ID,
							Owner:      conf.GlobalConfig.GatewayName,
							RuleID:     rule.ID,
							AgentID:    partner.ID,
							AccountID:  account.ID,
							LocalPath:  "file.loc",
							RemotePath: "file.rem",
							Start:      trans.Start.Local(),
							Status:     types.StatusPaused,
							Step:       types.StepData,
							Error:      types.TransferError{},
							Progress:   10,
							TaskNumber: 0,
						}

						var transfers model.Transfers
						So(db.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldNotBeEmpty)
						So(transfers[0], ShouldResemble, exp)
					})
				})
			})
		})
	})
}

func TestCancelTransfer(t *testing.T) {
	logger := log.NewLogger("rest_transfer_cancel_test")

	Convey("Testing the transfer resume handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		handler := cancelTransfer(nil)(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 planned transfer", func() {
			partner := &model.RemoteAgent{
				Name:        "test_server",
				Protocol:    testProto1,
				Address:     "localhost:1",
				ProtoConfig: json.RawMessage(`{}`),
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
				RuleID:     rule.ID,
				AgentID:    partner.ID,
				AccountID:  account.ID,
				LocalPath:  "file.loc",
				RemotePath: "file.rem",
				Start:      time.Date(2030, 1, 1, 1, 0, 0, 0, time.Local),
				Status:     types.StatusPlanned,
				Step:       types.StepNone,
				Error:      types.TransferError{},
				Progress:   0,
				TaskNumber: 0,
			}
			So(db.Insert(trans).Run(), ShouldBeNil)

			Convey("Given a request with the valid transfer ID parameter", func() {
				id := strconv.FormatUint(trans.ID, 10)
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

					Convey("Then the transfer should have been cancelled", func() {
						exp := model.HistoryEntry{
							ID:               trans.ID,
							Owner:            conf.GlobalConfig.GatewayName,
							RemoteTransferID: "",
							IsServer:         false,
							IsSend:           false,
							Account:          account.Login,
							Agent:            partner.Name,
							Protocol:         testProto1,
							LocalPath:        "file.loc",
							RemotePath:       "file.rem",
							Rule:             rule.Name,
							Start:            time.Date(2030, 1, 1, 1, 0, 0, 0, time.Local),
							Stop:             time.Time{},
							Status:           types.StatusCancelled,
							Error:            types.TransferError{},
							Step:             types.StepNone,
							Progress:         0,
							TaskNumber:       0,
						}

						var hist model.HistoryEntries
						So(db.Select(&hist).Run(), ShouldBeNil)
						So(hist, ShouldNotBeEmpty)
						So(hist[0], ShouldResemble, exp)
					})
				})
			})
		})
	})
}
