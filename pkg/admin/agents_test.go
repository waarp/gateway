package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

func agentListTest(handler http.Handler, db *database.Db, fieldName string,
	agent1, agent2, agent3, agent4 interface{}) {

	w := httptest.NewRecorder()
	expected := map[string][]interface{}{}

	check := func() {
		Convey("Then it should reply 'OK'", func() {
			So(w.Code, ShouldEqual, http.StatusOK)
		})

		Convey("Then the 'Content-Type' header should contain "+
			"'application/json'", func() {
			contentType := w.Header().Get("Content-Type")

			So(contentType, ShouldEqual, "application/json")
		})

		Convey("Then the response body should contain an array "+
			"of the requested agents in JSON format", func() {

			exp, err := json.Marshal(expected)

			So(err, ShouldBeNil)
			So(w.Body.String(), ShouldResemble, string(exp)+"\n")
		})
	}

	Convey("Given a database with 4 local agents", func() {
		err := db.Create(agent1)
		So(err, ShouldBeNil)
		err = db.Create(agent2)
		So(err, ShouldBeNil)
		err = db.Create(agent3)
		So(err, ShouldBeNil)
		err = db.Create(agent4)
		So(err, ShouldBeNil)

		Convey("Given a request with with no parameters", func() {
			r, err := http.NewRequest(http.MethodGet, "", nil)
			So(err, ShouldBeNil)

			Convey("When sending the request to the handler", func() {
				handler.ServeHTTP(w, r)

				expected[fieldName] = []interface{}{agent1, agent2, agent3, agent4}
				check()
			})
		})

		Convey("Given a request with a limit parameter", func() {
			r, err := http.NewRequest(http.MethodGet, "?limit=1", nil)
			So(err, ShouldBeNil)

			Convey("When sending the request to the handler", func() {
				handler.ServeHTTP(w, r)

				expected[fieldName] = []interface{}{agent1}
				check()
			})
		})

		Convey("Given a request with a offset parameter", func() {
			r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
			So(err, ShouldBeNil)

			Convey("When sending the request to the handler", func() {
				handler.ServeHTTP(w, r)

				expected[fieldName] = []interface{}{agent2, agent3, agent4}
				check()
			})
		})

		Convey("Given a request with a sort & order parameters", func() {
			r, err := http.NewRequest(http.MethodGet, "?sortby=protocol&order=desc", nil)
			So(err, ShouldBeNil)

			Convey("When sending the request to the handler", func() {
				handler.ServeHTTP(w, r)

				expected[fieldName] = []interface{}{agent1, agent2, agent3, agent4}
				check()
			})
		})

		Convey("Given a request with protocol parameters", func() {
			r, err := http.NewRequest(http.MethodGet, "?type=http&protocol=sftp", nil)
			So(err, ShouldBeNil)

			Convey("When sending the request to the handler", func() {
				handler.ServeHTTP(w, r)

				expected[fieldName] = []interface{}{agent1, agent2, agent3, agent4}
				check()
			})
		})
	})
}

func agentDeleteTest(handler http.Handler, db *database.Db, paramName, id string,
	existing interface{}) {

	w := httptest.NewRecorder()

	Convey("Given a request with the valid agent ID parameter", func() {
		r, err := http.NewRequest(http.MethodDelete, "", nil)
		So(err, ShouldBeNil)
		r = mux.SetURLVars(r, map[string]string{paramName: id})

		Convey("When sending the request to the handler", func() {
			handler.ServeHTTP(w, r)

			Convey("Then it should reply 'No Content'", func() {
				So(w.Code, ShouldEqual, http.StatusNoContent)
			})

			Convey("Then the body should be empty", func() {
				So(w.Body.String(), ShouldBeEmpty)
			})

			Convey("Then the agent should no longer be present in the database", func() {
				exist, err := db.Exists(existing)
				So(err, ShouldBeNil)
				So(exist, ShouldBeFalse)
			})
		})
	})

	Convey("Given a request with a non-existing agent ID parameter", func() {
		r, err := http.NewRequest(http.MethodDelete, "", nil)
		So(err, ShouldBeNil)
		r = mux.SetURLVars(r, map[string]string{paramName: "1000"})

		Convey("When sending the request to the handler", func() {
			handler.ServeHTTP(w, r)

			Convey("Then it should reply with a 'Not Found' error", func() {
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})
	})
}
