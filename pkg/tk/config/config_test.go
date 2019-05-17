package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestParse(t *testing.T) {

	Convey("Testing parsing", t, func() {
		Convey("Subsequent parsing overwrite values", func() {
			c := struct {
				Foo string `ini-name:"Foo"`
			}{
				Foo: "init",
			}

			b1 := strings.NewReader(`
Foo = blah
`)
			b2 := strings.NewReader(`
Foo = blih
`)

			p := NewParser(&c)

			err := p.Parse(b1)
			So(err, ShouldBeNil)
			err = p.Parse(b2)
			So(err, ShouldBeNil)

			So(c.Foo, ShouldEqual, "blih")
		})

		Convey("Defaults should be assigned if no value is given at initialization", func() {
			c := struct {
				Foo string `ini-name:"Foo" default:"default"`
			}{}

			b1 := strings.NewReader(`
`)

			p := NewParser(&c)

			err := p.Parse(b1)
			So(err, ShouldBeNil)

			So(c.Foo, ShouldEqual, "default")
		})

		Convey("An error is returned if the content is malformed", func() {
			c := struct {
				Foo string `ini-name:"Foo"`
			}{
				Foo: "init",
			}

			b1 := strings.NewReader(`
Foo blah
`)

			p := NewParser(&c)

			err := p.Parse(b1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "malformed")
		})

		Convey("An error is returned if values have the wrong type", func() {
			c := struct {
				Bar int `ini-name:"Bar"`
			}{
				Bar: 42,
			}

			b1 := strings.NewReader(`
Bar = string
`)

			p := NewParser(&c)

			err := p.Parse(b1)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid syntax")
		})

		Convey("Unknown keys are ignored", func() {
			c := struct {
				Foo string `ini-name:"Foo"`
			}{
				Foo: "init",
			}

			b1 := strings.NewReader(`
Foo = blah
Bar = baz
`)

			p := NewParser(&c)

			err := p.Parse(b1)
			So(err, ShouldBeNil)

			So(c.Foo, ShouldEqual, "blah")
		})

		Convey("Missing keys raise no errors", func() {
			c := struct {
				Foo string `ini-name:"Foo"`
				Bar string `ini-name:"Bar"`
			}{
				Foo: "init",
				Bar: "init2",
			}

			b1 := strings.NewReader(`
Foo = blah
`)

			p := NewParser(&c)

			err := p.Parse(b1)
			So(err, ShouldBeNil)

			So(c.Bar, ShouldEqual, "init2")
		})
	})
}

func writeConf(p *Parser) string {
	var buf bytes.Buffer
	p.Write(&buf)
	content := buf.String()
	return content
}

func TestWrite(t *testing.T) {

	type C struct {
		Foo string `ini-name:"Foo" description:"the foo description"`
		Bar string `ini-name:"Bar" default:"blah" description:"the bar description"`
	}

	Convey("Basic tests", t, func() {

		Convey("It can write conf", func() {
			c := C{
				Foo: "init",
			}

			p := NewParser(&c)

			content := writeConf(p)
			expected := `[global]
; the foo description
Foo = init

; the bar description
; Bar = blah

`

			So(content, ShouldEqual, expected)
		})
	})

	Convey("Write format", t, func() {

		Convey("The main section is called global", func() {
			c := C{
				Foo: "init",
			}

			p := NewParser(&c)

			content := writeConf(p)
			expected := "[global]"

			So(content, ShouldContainSubstring, expected)
		})

		Convey("It adds description as comments", func() {
			c := C{
				Foo: "init",
			}

			p := NewParser(&c)

			content := writeConf(p)
			expected := "; the foo description"

			So(content, ShouldContainSubstring, expected)
		})

		Convey("It writes the default value as a comment", func() {
			c := C{
				Foo: "init",
			}

			p := NewParser(&c)

			content := writeConf(p)
			expected := "; Bar = blah"

			So(content, ShouldContainSubstring, expected)
		})

		Convey("It writes user values without comments", func() {
			c := C{
				Foo: "init",
			}

			p := NewParser(&c)

			content := writeConf(p)
			expected := "Foo = init"

			So(content, ShouldContainSubstring, expected)
		})

		Convey("Options groups are separated in different sections", func() {
			type C struct {
				Group1 struct {
					Foo string `ini-name:"Foo"`
				} `group:"group1"`
				Group2 struct {
					Bar string `ini-name:"Bar" default:"blah"`
				} `group:"group2"`
			}
			c := C{}
			c.Group1.Foo = "init"

			p := NewParser(&c)

			content := writeConf(p)

			So(content, ShouldContainSubstring, "[group1]")
			So(content, ShouldContainSubstring, "[group2]")

		})

		Convey("Descriptions are written as comments in groups", func() {
			type C struct {
				Group1 struct {
					Foo string `ini-name:"Foo" description:"the foo description"`
				} `group:"group1"`
				Group2 struct {
					Bar string `ini-name:"Bar" default:"blah" description:"the bar description"`
				} `group:"group2"`
			}
			c := C{}
			c.Group1.Foo = "init"

			p := NewParser(&c)

			content := writeConf(p)

			So(content, ShouldContainSubstring, "; the foo description")
			So(content, ShouldContainSubstring, "; the bar description")

		})
	})

}

func TestUpdateFile(t *testing.T) {
	type C struct {
		Foo       string `ini-name:"Foo"`
		NewOption string `ini-name:"New"`
	}

	Convey("Given a configuration file", t, func() {
		content := []byte(`
[global]
Foo = bar
Old = baz
`)
		err := ioutil.WriteFile("config-test.ini", content, 0644)
		So(err, ShouldBeNil)

		Convey("When UpdateConf is called", func() {
			p := NewParser(&C{})
			err := p.UpdateFile("config-test.ini")
			So(err, ShouldBeNil)
			contentBytes, err := ioutil.ReadFile("config-test.ini")
			So(err, ShouldBeNil)
			content := string(contentBytes)

			Convey("Then existing options are kept with their values", func() {
				So(content, ShouldContainSubstring, "\nFoo = bar\n")
			})
			Convey("Then new options are added", func() {
				So(content, ShouldContainSubstring, "\n; New =\n")
			})
			Convey("Then OldOptions are removed", func() {
				So(content, ShouldNotContainSubstring, "\nOld =\n")
			})
		})

		os.Remove("config-test.ini")
	})
}
