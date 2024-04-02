package utils

import "code.waarp.fr/lib/r66"

// R66Hash returns the R66 "special hash" of the given password.
func R66Hash(pswd string) string {
	return string(r66.CryptPass([]byte(pswd)))
}
