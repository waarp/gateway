package admin

import (
	"net/http"
	"strconv"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

// contentSecurityPolicy is the content security policy of the administration
// service. Everything the web interface loads comes from the service itself:
// its scripts, its stylesheets and its fonts are files it serves, and it has
// no inline script. The images of a stylesheet may be embedded in it ("data:").
// Nothing may frame the interface, nor change the base of its links, nor send
// its forms elsewhere.
const contentSecurityPolicy = "default-src 'self'; img-src 'self' data:; object-src 'none'; " +
	"base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// securityHeaders tells the browser how to handle the answers of the
// administration service: no content sniffing, no framing by another site, a
// referrer kept to the service, a content security policy, and HTTPS only
// once the service was reached over TLS.
//
// It wraps the whole router rather than being one of its middlewares: the
// router only runs them for the requests that match a route, and the answers
// it gives on its own (not found, method not allowed) must carry the headers
// too.
func securityHeaders(config *conf.AdminConfig, next http.Handler) http.Handler {
	// An unset mode is the default one: the policy is reported, not enforced.
	var cspHeader string

	switch config.ContentSecurityPolicy {
	case conf.CSPOff:
	case conf.CSPEnforce:
		cspHeader = "Content-Security-Policy"
	default:
		cspHeader = "Content-Security-Policy-Report-Only"
	}

	// The header binds the whole host name: it is only sent once asked for,
	// and never with "includeSubDomains".
	var hsts string
	if maxAge := int64(config.HSTSMaxAge / time.Second); maxAge > 0 {
		hsts = "max-age=" + strconv.FormatInt(maxAge, 10)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "same-origin")

		if cspHeader != "" {
			header.Set(cspHeader, contentSecurityPolicy)
		}

		if hsts != "" && r.TLS != nil {
			header.Set("Strict-Transport-Security", hsts)
		}

		next.ServeHTTP(w, r)
	})
}
