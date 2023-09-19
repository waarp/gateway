package wg

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCloudGet(t *testing.T) {
	const (
		cloudName = "cloud_name"
		cloudType = "cloud_type"
		cloudKey  = "cloud_key"

		opt1, key1 = "opt1", "key1"
		opt2, key2 = "opt2", "key2"

		path = cloudsAPIPath + "/" + cloudName
	)

	Convey(`Testing the cloud "get" command`, t, func() {
		w := &strings.Builder{}
		command := &CloudGet{}

		respBody := map[string]any{
			"name":    cloudName,
			"type":    cloudType,
			"key":     cloudKey,
			"options": map[string]string{opt1: key1, opt2: key2},
		}
		expRequest := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}
		expResponse := &expectedResponse{
			status: http.StatusOK,
			body:   respBody,
		}

		testServer(t, expRequest, expResponse)

		Convey("When the command is executed", func() {
			_, err := flags.ParseArgs(command, []string{cloudName})
			So(err, ShouldBeNil)

			SoMsg("Then it should not return an error",
				command.execute(w), ShouldBeNil)
			SoMsg("Then it should print the cloud instance's info",
				w.String(),
				ShouldEqual,
				fmt.Sprintf(""+
					"── Cloud instance %q (%s)\n"+
					"   ├─ Key: %s\n"+
					"   ╰─ Options\n"+
					"      ├─ %s: %s\n"+
					"      ╰─ %s: %s\n",
					cloudName, cloudType,
					cloudKey,
					opt1, key1,
					opt2, key2,
				),
			)
		})
	})
}

func TestCloudAdd(t *testing.T) {
	const (
		cloudName   = "cloud_name"
		cloudType   = "cloud_type"
		cloudKey    = "cloud_key"
		cloudSecret = "cloud_secret"

		opt1, key1 = "opt1", "key1"
		opt2, key2 = "opt2", "key2"

		path     = "/api/clouds"
		location = path + "/" + cloudName
	)

	Convey(`Testing the cloud "add" command`, t, func() {
		w := &strings.Builder{}
		command := &CloudAdd{}

		reqBody := map[string]any{
			"name":    cloudName,
			"type":    cloudType,
			"key":     cloudKey,
			"secret":  cloudSecret,
			"options": map[string]any{opt1: key1, opt2: key2},
		}
		expRequest := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body:   reqBody,
		}
		expResponse := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		testServer(t, expRequest, expResponse)

		Convey("When the command is executed", func() {
			_, err := flags.ParseArgs(command, []string{
				"--name", cloudName,
				"--type", cloudType,
				"--key", cloudKey,
				"--secret", cloudSecret,
				"--options", fmt.Sprintf("%s:%s", opt1, key1),
				"--options", fmt.Sprintf("%s:%s", opt2, key2),
			})
			So(err, ShouldBeNil)

			SoMsg("Then it should not return an error",
				command.execute(w), ShouldBeNil)
			SoMsg("Then it should display a success message",
				w.String(),
				ShouldEqual,
				fmt.Sprintf("The cloud instance %q was successfully added.\n", cloudName),
			)
		})
	})
}

func TestCloudDelete(t *testing.T) {
	const (
		cloudName = "cloud_name"

		path = "/api/clouds/" + cloudName
	)

	Convey(`Testing the cloud "delete" command`, t, func() {
		w := &strings.Builder{}
		command := &CloudDelete{}

		expRequest := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		expResponse := &expectedResponse{status: http.StatusNoContent}

		testServer(t, expRequest, expResponse)

		Convey("When the command is executed", func() {
			_, err := flags.ParseArgs(command, []string{cloudName})
			So(err, ShouldBeNil)

			SoMsg("Then it should not return an error",
				command.execute(w), ShouldBeNil)
			SoMsg("Then it should display a success message",
				w.String(),
				ShouldEqual,
				fmt.Sprintf("The cloud instance %q was successfully deleted.\n", cloudName),
			)
		})
	})
}

func TestCloudUpdate(t *testing.T) {
	const (
		oldCloudName = "old_cloud_name"

		cloudName   = "cloud_name"
		cloudType   = "cloud_type"
		cloudKey    = "cloud_key"
		cloudSecret = "cloud_secret"

		opt1, key1 = "opt1", "key1"
		opt2, key2 = "opt2", "key2"

		path     = "/api/clouds/" + oldCloudName
		location = "/api/clouds/" + cloudName
	)

	Convey(`Testing the cloud "update" command`, t, func() {
		w := &strings.Builder{}
		command := &CloudUpdate{}

		reqBody := map[string]any{
			"name":    cloudName,
			"type":    cloudType,
			"key":     cloudKey,
			"secret":  cloudSecret,
			"options": map[string]any{opt1: key1, opt2: key2},
		}
		expRequest := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body:   reqBody,
		}
		expResponse := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		testServer(t, expRequest, expResponse)

		Convey("When the command is executed", func() {
			_, err := flags.ParseArgs(command, []string{
				oldCloudName,
				"--name", cloudName,
				"--type", cloudType,
				"--key", cloudKey,
				"--secret", cloudSecret,
				"--options", fmt.Sprintf("%s:%s", opt1, key1),
				"--options", fmt.Sprintf("%s:%s", opt2, key2),
			})
			So(err, ShouldBeNil)

			SoMsg("Then it should not return an error",
				command.execute(w), ShouldBeNil)
			SoMsg("Then it should display a success message",
				w.String(),
				ShouldEqual,
				fmt.Sprintf("The cloud instance %q was successfully updated.\n", cloudName),
			)
		})
	})
}

func TestCloudList(t *testing.T) {
	const (
		path = "/api/clouds"

		sort   = "name-"
		limit  = "10"
		offset = "5"

		cloud1     = "cloud1"
		cloud1Type = "cloud1_type"

		cloud2     = "cloud2"
		cloud2Type = "cloud2_type"
	)

	Convey(`Testing the cloud "list" command`, t, func() {
		w := &strings.Builder{}
		command := &CloudList{}

		expRequest := &expectedRequest{
			method: http.MethodGet,
			path:   path,
			values: url.Values{
				"sort":   {sort},
				"limit":  {limit},
				"offset": {offset},
			},
		}

		respBody := map[string]any{
			"clouds": []map[string]any{
				{"name": cloud1, "type": cloud1Type},
				{"name": cloud2, "type": cloud2Type},
			},
		}
		expResponse := &expectedResponse{
			status: http.StatusOK,
			body:   respBody,
		}

		testServer(t, expRequest, expResponse)

		Convey("When the command is executed", func() {
			_, err := flags.ParseArgs(command, []string{
				"--sort", sort,
				"--limit", limit,
				"--offset", offset,
			})
			So(err, ShouldBeNil)

			SoMsg("Then it should not return an error",
				command.execute(w), ShouldBeNil)

			SoMsg("Then it should display a list of the cloud instances",
				w.String(),
				ShouldEqual,
				fmt.Sprintf("Cloud instances:\n"+
					"╭─ Cloud instance %q (%s)\n"+
					"│  ├─ Key: <none>\n"+
					"│  ╰─ Options: <none>\n"+
					"╰─ Cloud instance %q (%s)\n"+
					"   ├─ Key: <none>\n"+
					"   ╰─ Options: <none>\n",
					cloud1, cloud1Type,
					cloud2, cloud2Type,
				),
			)
		})
	})
}
