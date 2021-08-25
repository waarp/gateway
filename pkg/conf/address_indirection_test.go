package conf

import (
	"bytes"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAddressOverrideParse(t *testing.T) {
	Convey("Given an address indirection configOverride parser", t, func() {
		over := &addressOverride{
			addressMap:   map[string]string{"192.168.1.1": "waarp.org"},
			Indirections: []string{"192.168.1.1 -> waarp.org"},
		}

		Convey("When parsing an address value", func() {
			type testCase struct {
				desc, value string
				check       func(*testCase, error)
			}

			shouldSucceed := func(key, val string) func(*testCase, error) {
				return func(t *testCase, err error) {
					Convey("Then it should have correctly split the addresses", func() {
						So(err, ShouldBeNil)
						So(over.addressMap[key], ShouldEqual, val)
					})
				}
			}
			shouldFailWith := func(msg string) func(*testCase, error) {
				return func(_ *testCase, err error) {
					Convey("Then it should have failed to split the addresses", func() {
						So(err, ShouldBeError, msg)
					})
				}
			}

			testCases := []testCase{
				{"2 IPv4 addresses", "127.0.0.1 -> 0.0.0.0", shouldSucceed("127.0.0.1", "0.0.0.0")},
				{"an IPv4 & an IPv6 address", "127.0.0.1 -> [::1]", shouldSucceed("127.0.0.1", "[::1]")},
				{"an IPv4 address & a hostname", "127.0.0.1 -> example.com", shouldSucceed("127.0.0.1", "example.com")},
				{"2 IPv6 addresses", "[::1] -> [2620:fe::fe]", shouldSucceed("[::1]", "[2620:fe::fe]")},
				{"an IPv6 & an IPv4 address", "[::1] -> 127.0.0.1", shouldSucceed("[::1]", "127.0.0.1")},
				{"an IPv6 & a hostname", "[::1] -> example.com", shouldSucceed("[::1]", "example.com")},
				{"2 hostnames", "example.com -> waarp.fr", shouldSucceed("example.com", "waarp.fr")},
				{"a hostname & an IPv4 address", "example.com -> 127.0.0.1", shouldSucceed("example.com", "127.0.0.1")},
				{"a hostname & an IPv6 address", "example.com -> [::1]", shouldSucceed("example.com", "[::1]")},
				{"that the '->' separator is missing", "127.0.0.1 0.0.0.0", shouldFailWith(
					"malformed address indirection '127.0.0.1 0.0.0.0' (missing '->' separator)")},
				{"that there are too many '->' separators", "127.0.0.1 -> 0.0.0.0 ->", shouldFailWith(
					"malformed address indirection '127.0.0.1 -> 0.0.0.0 ->' (too many '->' separators)")},
				{"that the target address is a duplicate", "192.168.1.1 -> 0.0.0.0", shouldFailWith(
					"duplicate address indirection target '192.168.1.1'")},
			}

			for _, test := range testCases {
				Convey("Given "+test.desc, func() {
					over.Indirections = append(over.Indirections, test.value)
					err := over.parse()
					test.check(&test, err)
				})
			}
		})
	})
}

func TestOverrideWrite(t *testing.T) {
	Convey("Given an configOverride parser", t, func() {
		over := configOverride{
			ListenAddresses: &addressOverride{
				Indirections: []string{
					"192.168.1.1 -> waarp.org",
					"[::1] -> 0.0.0.0",
					"localhost -> 127.0.0.1",
				},
			},
		}

		Convey("When writing the configOverride settings", func() {
			buf := bytes.Buffer{}
			over.writeTo(&buf)

			Convey("Then it should have properly written the configOverride settings", func() {
				nextLine := func() string {
					line, err := buf.ReadBytes('\n')
					So(err, ShouldBeNil)
					return string(line)
				}
				line1 := "IndirectAddress = 192.168.1.1 -> waarp.org\n"
				line2 := "IndirectAddress = [::1] -> 0.0.0.0\n"
				line3 := "IndirectAddress = localhost -> 127.0.0.1\n"

				So(nextLine(), ShouldEqual, "[Address Indirection]\n")
				So(nextLine(), ShouldEqual, "; Replace the target address with another one\n")
				So(buf.String(), ShouldBeIn,
					line1+line2+line3+"\n", line1+line3+line2+"\n",
					line2+line1+line3+"\n", line2+line3+line1+"\n",
					line3+line1+line2+"\n", line3+line2+line1+"\n")
			})
		})
	})
}

func TestAddIndirection(t *testing.T) {
	Convey("Given an address configOverride", t, func(c C) {
		LocalOverrides = &configOverride{
			filename: testhelpers.TempFile(c, "test_override_add_indirection_*.ini"),
			ListenAddresses: &addressOverride{
				addressMap: map[string]string{
					"localhost:5555": "127.0.0.1:8080",
				},
			},
		}

		Convey("When adding a new indirection", func() {
			So(AddIndirection("1.2.3.4", "9.8.7.6"), ShouldBeNil)

			Convey("Then it should have added the indirection", func() {
				So(LocalOverrides.ListenAddresses.addressMap["1.2.3.4"],
					ShouldEqual, "9.8.7.6")
				So(LocalOverrides.ListenAddresses.Indirections, ShouldContain,
					"1.2.3.4 -> 9.8.7.6")
				So(LocalOverrides.ListenAddresses.Indirections, ShouldContain,
					"localhost:5555 -> 127.0.0.1:8080")
			})
		})

		Convey("When updating an existing indirection", func() {
			So(AddIndirection("localhost:5555", "9.8.7.6"), ShouldBeNil)

			Convey("Then it should have updated the indirection", func() {
				So(LocalOverrides.ListenAddresses.addressMap["localhost:5555"],
					ShouldEqual, "9.8.7.6")
				So(LocalOverrides.ListenAddresses.Indirections, ShouldContain,
					"localhost:5555 -> 9.8.7.6")
			})
		})
	})
}

func TestRemoveIndirection(t *testing.T) {
	Convey("Given an address configOverride", t, func(c C) {
		LocalOverrides = &configOverride{
			filename: testhelpers.TempFile(c, "test_override_remove_indirection_*.ini"),
			ListenAddresses: &addressOverride{
				addressMap: map[string]string{
					"localhost:5555": "127.0.0.1:8080",
					"1.2.3.4":        "9.8.7.6",
				},
			},
		}

		Convey("When removing an existing indirection", func() {
			So(RemoveIndirection("localhost:5555"), ShouldBeNil)

			Convey("Then it should have removed the indirection", func() {
				So(LocalOverrides.ListenAddresses.addressMap["1.2.3.4"],
					ShouldEqual, "9.8.7.6")
				So(LocalOverrides.ListenAddresses.Indirections, ShouldResemble,
					[]string{"1.2.3.4 -> 9.8.7.6"})
			})
		})

		Convey("When removing a non-existing indirection", func() {
			So(RemoveIndirection("unknown"), ShouldBeNil)

			Convey("Then it should not have deleted anything", func() {
				So(LocalOverrides.ListenAddresses.Indirections, ShouldContain,
					"localhost:5555 -> 127.0.0.1:8080")
				So(LocalOverrides.ListenAddresses.Indirections, ShouldContain,
					"1.2.3.4 -> 9.8.7.6")
			})
		})
	})
}

func TestGetRealAddress(t *testing.T) {
	Convey("Given an address configOverride", t, func() {
		LocalOverrides = &configOverride{
			ListenAddresses: &addressOverride{
				addressMap: map[string]string{
					"localhost:5555": "127.0.0.1:8080",
					"1.2.3.4":        "9.8.7.6",
				},
			},
		}

		Convey("Given a full address match", func() {
			redirect, err := GetRealAddress("localhost:5555")
			So(err, ShouldBeNil)

			Convey("Then it should return the associated address", func() {
				So(redirect, ShouldEqual, "127.0.0.1:8080")
			})
		})

		Convey("Given a host match", func() {
			redirect, err := GetRealAddress("1.2.3.4:5555")
			So(err, ShouldBeNil)

			Convey("Then it should return the associated host with the old port", func() {
				So(redirect, ShouldEqual, "9.8.7.6:5555")
			})
		})

		Convey("Given no match", func() {
			redirect, err := GetRealAddress("192.168.1.1:6666")
			So(err, ShouldBeNil)

			Convey("Then it should return the address as is", func() {
				So(redirect, ShouldEqual, "192.168.1.1:6666")
			})
		})

		Convey("Given that the address is incorrectly formatted", func() {
			_, err := GetRealAddress("waarp.fr")

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, "address waarp.fr: missing port in address")
			})
		})
	})
}
