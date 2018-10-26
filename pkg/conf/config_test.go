package conf

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLoadServerConfig(t *testing.T) {
	directContent := []byte(`[Log]
LogTo = direct-foo
Level = direct-bar

`)
	parentEtcContent := []byte(`[Log]
LogTo = parent-etc-foo
SyslogFacility = parent-etc-baz

`)
	userConfContent := []byte(`[Log]
LogTo = user-foo

`)
	badContent := []byte(`[Log]
LogTo user-foo

`)

	Convey("Given the user did not pass a configuration file", t, func() {
		userConf := ""

		Convey("Given there are no configuration file", func() {

			Convey("When the configuration is loaded", func() {
				c, err := LoadServerConfig(userConf)

				Convey("Then there are no file not found error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then the default configuration is used", func() {
					So(c.Log.LogTo, ShouldEqual, "stdout")
				})

			})

		})

		Convey("Given a configuration file in the same folder", func() {
			Convey("Given it can be parsed", func() {
				err := ioutil.WriteFile("gatewayd.ini", directContent, 0644)
				So(err, ShouldBeNil)

				Convey("When the configuration is loaded", func() {
					c, err := LoadServerConfig(userConf)

					Convey("Then there are no file not found error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it is parsed", func() {
						So(c.Log.LogTo, ShouldEqual, "direct-foo")
					})

				})

				So(os.Remove("gatewayd.ini"), ShouldBeNil)
			})

			Convey("Given it cannot be parsed", func() {
				err := ioutil.WriteFile("gatewayd.ini", badContent, 0644)
				So(err, ShouldBeNil)

				Convey("When the configuration is loaded", func() {
					_, err := LoadServerConfig(userConf)

					Convey("Then an error is returned", func() {
						So(err, ShouldNotBeNil)
					})

				})

				So(os.Remove("gatewayd.ini"), ShouldBeNil)

			})

		})

		Convey("Given a configuration file in the etc folder", func() {
			err := os.Mkdir("etc", 0755)
			So(err, ShouldBeNil)
			err = ioutil.WriteFile("etc/gatewayd.ini", parentEtcContent, 0644)
			So(err, ShouldBeNil)

			Convey("When the configuration is loaded", func() {
				c, err := LoadServerConfig(userConf)

				Convey("Then there are no file not found error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then it is parsed", func() {
					So(c.Log.LogTo, ShouldEqual, "parent-etc-foo")
				})

			})

			So(os.RemoveAll("etc"), ShouldBeNil)

		})

		Convey("Given both configuration files exist", func() {
			err := ioutil.WriteFile("gatewayd.ini", directContent, 0644)
			So(err, ShouldBeNil)
			err = os.Mkdir("etc", 0755)
			So(err, ShouldBeNil)
			err = ioutil.WriteFile("etc/gatewayd.ini", parentEtcContent, 0644)
			So(err, ShouldBeNil)

			Convey("When the configuration is loaded", func() {
				c, err := LoadServerConfig(userConf)

				Convey("Then there are no file not found error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then only the one in the same directory is parsed", func() {
					So(c.Log.LogTo, ShouldEqual, "direct-foo")
					So(c.Log.SyslogFacility, ShouldEqual, "local0")
				})

			})

			So(os.RemoveAll("etc"), ShouldBeNil)
			So(os.Remove("gatewayd.ini"), ShouldBeNil)

		})

	})

	Convey("Given the user passed a configuration file", t, func() {
		userConf := "userConf.ini"

		Convey("Given that file exist", func() {
			err := ioutil.WriteFile(userConf, userConfContent, 0644)
			So(err, ShouldBeNil)

			Convey("Given no other configuration files exist", func() {

				Convey("When the configuration is loaded", func() {
					c, err := LoadServerConfig(userConf)

					Convey("Then there are no file not found error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it is parsed", func() {
						So(c.Log.LogTo, ShouldEqual, "user-foo")
					})

				})

			})

			Convey("Given both other configuration files exist", func() {
				err := ioutil.WriteFile("gatewayd.ini", directContent, 0644)
				So(err, ShouldBeNil)
				err = os.Mkdir("etc", 0755)
				So(err, ShouldBeNil)
				err = ioutil.WriteFile("etc/gatewayd.ini", parentEtcContent, 0644)
				So(err, ShouldBeNil)

				Convey("When the configuration is loaded", func() {
					c, err := LoadServerConfig(userConf)

					Convey("Then there are no file not found error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it is parsed", func() {
						So(c.Log.LogTo, ShouldEqual, "user-foo")
					})

					Convey("Then the other configuration files are not used", func() {
						So(c.Log.Level, ShouldEqual, "INFO")
						So(c.Log.SyslogFacility, ShouldEqual, "local0")
					})

				})

				So(os.RemoveAll("etc"), ShouldBeNil)
				So(os.Remove("gatewayd.ini"), ShouldBeNil)

			})

			So(os.Remove(userConf), ShouldBeNil)

		})

		Convey("Given that cannot be parsed", func() {
			err := ioutil.WriteFile(userConf, badContent, 0644)
			So(err, ShouldBeNil)

			Convey("When the configuration is loaded", func() {
				_, err := LoadServerConfig(userConf)

				Convey("Then an error is returned", func() {
					So(err, ShouldNotBeNil)
				})

			})

			So(os.Remove(userConf), ShouldBeNil)

		})

		Convey("Given that file does not exist", func() {
			userConf := "does-not-exist.ini"

			Convey("When the configuration is loaded", func() {
				_, err := LoadServerConfig(userConf)

				Convey("Then an error is returned", func() {
					So(err, ShouldNotBeNil)
				})

			})

		})

	})

}

func TestUpdateServerConfig(t *testing.T) {
	oldContent := []byte(`[global]
; old desc
; LogTo = old-default-foo
Bar = direct-bar
HasBeenRemoved = true

`)
	getNewContent := func(configFile string) string {
		content, err := ioutil.ReadFile(configFile)
		if err != nil {
			panic(err.Error())
		}
		return string(content)
	}

	Convey("Given the user passed a configuration file", t, func() {
		userConfig := "userConfig.ini"

		Convey("Given it can be read", func() {
			err := ioutil.WriteFile(userConfig, oldContent, 0644)
			So(err, ShouldBeNil)

			Convey("When the file is updated", func() {
				err := UpdateServerConfig(userConfig)
				newContent := getNewContent(userConfig)

				Convey("Then there are no errors", func() {
					So(err, ShouldBeNil)
				})

				Convey("And the old values are removed", func() {
					So(newContent, ShouldNotContainSubstring, "HasBeenRemoved")
				})

				Convey("And the new values are added", func() {
					So(newContent, ShouldContainSubstring, "SyslogFacility")
				})

				Convey("And default values are changed if needed", func() {
					So(newContent, ShouldNotContainSubstring, "; LogTo = old-default-foo")
					So(newContent, ShouldContainSubstring, "; LogTo = stdout")
				})

				Convey("And comments are updated if needed", func() {
					So(newContent, ShouldNotContainSubstring, "; old desc")
					So(newContent, ShouldContainSubstring, "; All messages")
				})

			})

			So(os.Remove(userConfig), ShouldBeNil)

		})

		Convey("Given it cannot be read", func() {

			Convey("When the file is updated", func() {
				err := UpdateServerConfig(userConfig)

				Convey("Then an error is returned", func() {
					So(err, ShouldNotBeNil)
				})

			})

		})

		Convey("Given it cannot be parsed", func() {
			badContent := bytes.Replace(oldContent, []byte("="), []byte(""), -1)
			err := ioutil.WriteFile(userConfig, badContent, 0644)
			So(err, ShouldBeNil)

			Convey("When the file is updated", func() {
				err := UpdateServerConfig(userConfig)

				Convey("Then an error is returned", func() {
					So(err, ShouldNotBeNil)
				})

			})

			So(os.Remove(userConfig), ShouldBeNil)

		})

	})

}
