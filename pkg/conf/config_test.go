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

[Admin]
Host = direct-address

`)
	parentEtcContent := []byte(`[Log]
LogTo = parent-etc-foo
SyslogFacility = parent-etc-baz

[Admin]
Host = parent-etc-address

`)
	userConfContent := []byte(`[Log]
LogTo = user-foo

[Admin]
Host = user-address

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
					So(c.Admin.Host, ShouldEqual, "localhost")
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
						So(c.Admin.Host, ShouldEqual, "direct-address")
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
			err := os.Mkdir("etc", 0750)
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
					So(c.Admin.Host, ShouldEqual, "parent-etc-address")
				})

			})

			So(os.RemoveAll("etc"), ShouldBeNil)

		})

		Convey("Given both configuration files exist", func() {
			err := ioutil.WriteFile("gatewayd.ini", directContent, 0644)
			So(err, ShouldBeNil)
			err = os.Mkdir("etc", 0750)
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
					So(c.Admin.Host, ShouldEqual, "direct-address")
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
						So(c.Admin.Host, ShouldEqual, "user-address")
					})

				})

			})

			Convey("Given both other configuration files exist", func() {
				err := ioutil.WriteFile("gatewayd.ini", directContent, 0644)
				So(err, ShouldBeNil)
				err = os.Mkdir("etc", 0750)
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
						So(c.Admin.Host, ShouldEqual, "user-address")
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

func TestNormalizePaths(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to retrieve working directory: %s", err)
	}

	Convey("Given a conf object", t, func() {
		conf := &ServerConfig{}

		testCases := []struct {
			desc                            string
			home, in, out, work             string
			expDesc                         string
			expHome, expIn, expOut, expWork string
		}{
			{"all paths are absolute", "/home", "/home/in", "/home/out", "/home/work",
				"the paths should be unchanged", "/home", "/home/in", "/home/out", "/home/work"},
			{"the sub dirs are relative", "/home", "in", "out", "work",
				"the paths should be placed under the home dir", "/home", "/home/in", "/home/out", "/home/work"},
			{"the home dir is relative", "./home", "/in", "/out", "/work",
				"the home dir should be under the current dir", wd + "/home", "/in", "/out", "/work"},
			{"the home dir is empty", "", "/in", "/out", "/work",
				"the home dir should be the current dir", wd, "/in", "/out", "/work"},
		}

		for _, testCase := range testCases {
			Convey("Given that "+testCase.desc, func() {
				conf.Paths.GatewayHome = mkWin(testCase.home)
				conf.Paths.InDirectory = mkWin(testCase.in)
				conf.Paths.OutDirectory = mkWin(testCase.out)
				conf.Paths.WorkDirectory = mkWin(testCase.work)

				Convey("When normalizing the paths", func() {
					So(normalizePaths(conf), ShouldBeNil)

					Convey("Then "+testCase.expDesc, func() {
						So(conf.Paths.GatewayHome, ShouldEqual, mkWin(testCase.expHome))
						So(conf.Paths.InDirectory, ShouldEqual, mkWin(testCase.expIn))
						So(conf.Paths.OutDirectory, ShouldEqual, mkWin(testCase.expOut))
						So(conf.Paths.WorkDirectory, ShouldEqual, mkWin(testCase.expWork))
					})
				})
			})
		}
	})
}
