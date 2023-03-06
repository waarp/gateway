package rest

import (
	"fmt"
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	clientsPathFormat = "/api/clients"
	clientPathFormat  = "/api/clients/%s"
)

func TestClientAdd(t *testing.T) {
	Convey("Given a REST server", t, func(c C) {
		test := makeTestRESTServer(c)

		Convey("When adding a new client", func() {
			jsonClient := map[string]any{
				"name":         "new_client",
				"protocol":     testProto1,
				"protoConfig":  map[string]any{"key": "val"},
				"localAddress": ":0",
			}

			url := fmt.Sprintf(test.URL + clientsPathFormat)
			resp := test.post(url, jsonClient)

			Convey("Then it should return a code 201", func() {
				So(readBody(resp.Body), ShouldBeBlank)
				So(resp.StatusCode, ShouldEqual, http.StatusCreated)

				Convey("Then it should have added the client to the database", func() {
					var dbClients model.Clients
					So(test.db.Select(&dbClients).Run(), ShouldBeNil)
					So(dbClients, ShouldHaveLength, 1)
					So(dbClients[0], ShouldResemble, &model.Client{
						ID:           1,
						Owner:        conf.GlobalConfig.GatewayName,
						Name:         jsonClient["name"].(string),
						Protocol:     jsonClient["protocol"].(string),
						LocalAddress: jsonClient["localAddress"].(string),
						ProtoConfig:  jsonClient["protoConfig"].(map[string]any),
					})
				})
			})
		})
	})
}

func TestClientList(t *testing.T) {
	Convey("Given a REST server", t, func(c C) {
		test := makeTestRESTServer(c)

		dbClient1 := &model.Client{
			Name:         "test_client1",
			Protocol:     testProto1,
			LocalAddress: ":1",
		}
		So(test.db.Insert(dbClient1).Run(), ShouldBeNil)

		dbClient2 := &model.Client{
			Name:         "test_client2",
			Protocol:     testProto2,
			LocalAddress: ":2",
		}
		So(test.db.Insert(dbClient2).Run(), ShouldBeNil)

		Convey("When listing the clients", func() {
			url := fmt.Sprintf(test.URL + clientsPathFormat)
			resp := test.get(url)

			Convey("Then it should return a code 200", func() {
				So(resp.StatusCode, ShouldEqual, http.StatusOK)
				So(resp.Header.Get("Content-Type"), ShouldEqual, "application/json")

				Convey("Then it should have returned the clients", func() {
					content := parseBody(resp.Body)
					expected := map[string]any{
						"clients": []any{
							map[string]any{
								"name":         dbClient1.Name,
								"protocol":     dbClient1.Protocol,
								"localAddress": dbClient1.LocalAddress,
								"protoConfig":  dbClient1.ProtoConfig,
							},
							map[string]any{
								"name":         dbClient2.Name,
								"protocol":     dbClient2.Protocol,
								"localAddress": dbClient2.LocalAddress,
								"protoConfig":  dbClient2.ProtoConfig,
							},
						},
					}

					So(content, ShouldResemble, expected)
				})
			})
		})
	})
}

func TestClientGet(t *testing.T) {
	Convey("Given a REST server", t, func(c C) {
		test := makeTestRESTServer(c)

		dbClient := &model.Client{
			Name:         "test_client",
			Protocol:     testProto1,
			LocalAddress: ":1",
		}
		So(test.db.Insert(dbClient).Run(), ShouldBeNil)

		Convey("When retrieving a client", func() {
			url := fmt.Sprintf(test.URL+clientPathFormat, dbClient.Name)
			resp := test.get(url)

			Convey("Then it should return a code 200", func() {
				So(resp.StatusCode, ShouldEqual, http.StatusOK)
				So(resp.Header.Get("Content-Type"), ShouldEqual, "application/json")

				Convey("Then it should have returned the client", func() {
					content := parseBody(resp.Body)
					expected := map[string]any{
						"name":         dbClient.Name,
						"protocol":     dbClient.Protocol,
						"localAddress": dbClient.LocalAddress,
						"protoConfig":  dbClient.ProtoConfig,
					}

					So(content, ShouldResemble, expected)
				})
			})
		})
	})
}

func TestClientUpdate(t *testing.T) {
	Convey("Given a REST server", t, func(c C) {
		test := makeTestRESTServer(c)

		dbClient := &model.Client{
			Name:         "test_client",
			Protocol:     testProto1,
			LocalAddress: ":1",
		}
		So(test.db.Insert(dbClient).Run(), ShouldBeNil)

		Convey("When updating a client", func() {
			jsonClient := map[string]any{
				"name":         "new_client",
				"protocol":     testProto2,
				"localAddress": ":2",
			}

			url := fmt.Sprintf(test.URL+clientPathFormat, dbClient.Name)
			resp := test.patch(url, jsonClient)

			Convey("Then it should return a code 201", func() {
				So(readBody(resp.Body), ShouldBeBlank)
				So(resp.StatusCode, ShouldEqual, http.StatusCreated)

				Convey("Then it should have updated the client", func() {
					var dbClients model.Clients
					So(test.db.Select(&dbClients).Run(), ShouldBeNil)
					So(dbClients, ShouldHaveLength, 1)
					So(dbClients[0], ShouldResemble, &model.Client{
						ID:           dbClient.ID,
						Owner:        dbClient.Owner,
						Name:         jsonClient["name"].(string),
						Protocol:     jsonClient["protocol"].(string),
						LocalAddress: jsonClient["localAddress"].(string),
						ProtoConfig:  dbClient.ProtoConfig,
					})
				})
			})
		})
	})
}

func TestClientReplace(t *testing.T) {
	Convey("Given a REST server", t, func(c C) {
		test := makeTestRESTServer(c)

		dbClient := &model.Client{
			Name:         "test_client",
			Protocol:     testProto1,
			LocalAddress: ":1",
		}
		So(test.db.Insert(dbClient).Run(), ShouldBeNil)

		Convey("When replacing a client", func() {
			jsonClient := map[string]any{
				"name":         "new_client",
				"protocol":     testProto2,
				"localAddress": ":2",
			}

			url := fmt.Sprintf(test.URL+clientPathFormat, dbClient.Name)
			resp := test.put(url, jsonClient)

			Convey("Then it should return a code 201", func() {
				So(readBody(resp.Body), ShouldBeBlank)
				So(resp.StatusCode, ShouldEqual, http.StatusCreated)

				Convey("Then it should have updated the client", func() {
					var dbClients model.Clients
					So(test.db.Select(&dbClients).Run(), ShouldBeNil)
					So(dbClients, ShouldHaveLength, 1)
					So(dbClients[0], ShouldResemble, &model.Client{
						ID:           dbClient.ID,
						Owner:        dbClient.Owner,
						Name:         jsonClient["name"].(string),
						Protocol:     jsonClient["protocol"].(string),
						LocalAddress: jsonClient["localAddress"].(string),
						ProtoConfig:  map[string]any{},
					})
				})
			})
		})
	})
}

func TestClientDelete(t *testing.T) {
	Convey("Given a REST server", t, func(c C) {
		test := makeTestRESTServer(c)

		dbClient := &model.Client{
			Name:         "test_client",
			Protocol:     testProto1,
			LocalAddress: ":1",
		}
		So(test.db.Insert(dbClient).Run(), ShouldBeNil)

		dbClient2 := &model.Client{
			Name:         "test_client2",
			Protocol:     testProto2,
			LocalAddress: ":2",
		}
		So(test.db.Insert(dbClient2).Run(), ShouldBeNil)

		Convey("When deleting the client", func() {
			url := fmt.Sprintf(test.URL+clientPathFormat, dbClient.Name)
			resp := test.delete(url)

			Convey("Then it should return a code 204", func() {
				So(readBody(resp.Body), ShouldBeBlank)
				So(resp.StatusCode, ShouldEqual, http.StatusNoContent)

				Convey("Then it should have deleted the client", func() {
					var dbClients model.Clients
					So(test.db.Select(&dbClients).Run(), ShouldBeNil)
					So(dbClients, ShouldHaveLength, 1)
					So(dbClients[0], ShouldResemble, dbClient2)
				})
			})
		})
	})
}
