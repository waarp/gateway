package protoform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A number field tells the browser what it accepts, and reads what the browser
// (or a client without JavaScript) sends with the same bounds.
func TestNumber(t *testing.T) {
	t.Parallel()

	translate := translator(t)

	t.Run("Given a whole number", func(t *testing.T) {
		t.Parallel()

		port := Number{Min: 1, Max: 65535}
		assert.Equal(t, "numeric", port.InputMode())
		assert.Equal(t, "1", port.StepAttr(), "whole numbers go one by one")
		assert.Equal(t, "1", port.MinAttr())
		assert.Equal(t, "65535", port.MaxAttr())

		value, msg := port.Read(translate, " 8080 ")
		assert.Empty(t, msg)
		assert.InDelta(t, 8080, value, 0)

		_, msg = port.Read(translate, "70000")
		assert.Equal(t, "Indiquez un nombre entier entre 1 et 65535.", msg)

		_, msg = port.Read(translate, "1.5")
		assert.Equal(t, "Indiquez un nombre entier entre 1 et 65535.", msg, "a decimal is not a whole number")

		attempts := Number{Max: 127, Step: 10}
		assert.Equal(t, "10", attempts.StepAttr(), "a step the description sets")

		_, msg = attempts.Read(translate, "-1")
		assert.Equal(t, "Indiquez un nombre entier entre 0 et 127.", msg)

		free := Number{}
		assert.Equal(t, "0", free.MinAttr())
		assert.Equal(t, "", free.MaxAttr(), "no maximum")

		for _, raw := range []string{"abc", "-3", "1e3", ""} {
			_, msg = free.Read(translate, raw)
			assert.Equalf(t, "Indiquez un nombre entier, 0 ou plus.", msg, "%q is refused", raw)
		}

		value, msg = free.Read(translate, "007")
		assert.Empty(t, msg)
		assert.Equal(t, "7", NumberText(value), "stored as the product reads it")
	})

	t.Run("Given a decimal number", func(t *testing.T) {
		t.Parallel()

		factor := Number{Decimal: true}
		assert.Equal(t, "decimal", factor.InputMode(), "the keyboard of a phone offers the separator")
		assert.Equal(t, "any", factor.StepAttr(), "any precision")
		assert.Equal(t, "0", factor.MinAttr())
		assert.Equal(t, "", factor.MaxAttr())

		value, msg := factor.Read(translate, "1,5")
		assert.Empty(t, msg)
		assert.Equal(t, "1.5", NumberText(value), "a decimal comma is accepted and stored with a dot")

		value, msg = factor.Read(translate, "2.0")
		assert.Empty(t, msg)
		assert.Equal(t, "2", NumberText(value))

		value, msg = factor.Read(translate, "0")
		assert.Empty(t, msg)
		assert.InDelta(t, 0, value, 0, "the minimum is accepted")

		for _, raw := range []string{"-1", "deux", "1,5,2", "inf", "NaN", "1e400", ""} {
			_, msg = factor.Read(translate, raw)
			assert.Equalf(t, "Indiquez un nombre, 0 ou plus (1,5 par exemple).", msg, "%q is refused", raw)
		}

		share := Number{Decimal: true, Min: 0.5, Max: 2.5, Step: 0.1}
		assert.Equal(t, "0.1", share.StepAttr())
		assert.Equal(t, "0.5", share.MinAttr())
		assert.Equal(t, "2.5", share.MaxAttr())

		_, msg = share.Read(translate, "3")
		assert.Equal(t, "Indiquez un nombre entre 0,5 et 2,5.", msg, "the bounds are written as the language reads them")

		value, msg = share.Read(translate, "1.25")
		assert.Empty(t, msg)
		assert.InDelta(t, 1.25, value, 0, "the step is the browser's business, not a bound")

		_, msg = (&Number{Decimal: true, Max: 10}).Read(translate, "x")
		assert.Equal(t, "Indiquez un nombre entre 0 et 10.", msg)

		_, msg = (&Number{Decimal: true, Min: 1}).Read(translate, "0.5")
		assert.Equal(t, "Indiquez un nombre, 1 ou plus (1,5 par exemple).", msg)
	})
}
