package types

import (
	"fmt"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAddress(t *testing.T) {
	t.Parallel()

	const (
		host = "localhost"
		port = 8080
	)

	t.Run("Given a valid address", func(t *testing.T) {
		t.Parallel()

		address, err := NewAddress(fmt.Sprintf("%s:%d", host, port))
		require.NoError(t, err, "then it should not return an error")

		assert.Equal(t, host, address.Host, "then the host should match")
		assert.EqualValues(t, port, address.Port, "then the port should match")
	})

	t.Run("Given an invalid address", func(t *testing.T) {
		t.Parallel()

		t.Run("Given an address the wrong format", func(t *testing.T) {
			t.Parallel()

			var addrErr *net.AddrError

			_, err := NewAddress(host)
			require.ErrorAs(t, err, &addrErr,
				"then it should return an address error")
			assert.Equal(t, "missing port in address", addrErr.Err,
				"then it should return a missing port error")
		})

		t.Run("Given an address with an invalid port number", func(t *testing.T) {
			t.Parallel()

			_, err := NewAddress(fmt.Sprintf("%s:99999", host))
			require.ErrorIs(t, err, strconv.ErrRange,
				"then it should return a number parsing error")
		})
	})
}

func TestAddressString(t *testing.T) {
	t.Parallel()

	t.Run("Given a valid address", func(t *testing.T) {
		t.Parallel()

		address := &Address{Host: "localhost", Port: 8080}

		assert.Equal(t,
			fmt.Sprintf("%s:%d", address.Host, address.Port),
			address.String(),
			"then the string should match")
	})

	t.Run("Given a zero address", func(t *testing.T) {
		t.Parallel()

		address := &Address{}

		assert.Equal(t, "", address.String(),
			"then the string should be empty")
	})
}

func TestAddressValidate(t *testing.T) {
	t.Parallel()

	const (
		host = "127.0.0.1"
		port = 8080
	)

	t.Run("Given a valid address", func(t *testing.T) {
		t.Parallel()

		address := &Address{Host: host, Port: port}

		require.NoError(t, address.Validate(), "then it should not return an error")
	})

	t.Run("Given an invalid address", func(t *testing.T) {
		t.Parallel()

		address := &Address{Host: "not_a_host", Port: port}

		var addrErr *net.DNSError

		require.ErrorAs(t, address.Validate(), &addrErr,
			"then it should return a DNS error")
		assert.Equal(t, "no such host", addrErr.Err,
			"then it should return an invalid host error")
	})
}
