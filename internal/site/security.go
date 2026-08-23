package site

import (
	"net/http"
	"os"
)

// contentSecurityPolicy locks the page down to the handful of things this site
// actually loads. Everything is denied by default and re-enabled per directive.
//
// No directive carries an 'unsafe-' token. All JS ships as external modules
// under /static, and the templates set no style="..." attributes, so neither
// inline scripts nor inline styles need whitelisting. Keep it that way: putting
// a style attribute back in a template means loosening style-src for the whole
// site, which is what CSP is here to prevent.
const contentSecurityPolicy = "default-src 'none'; " +
	"script-src 'self'; " +
	"style-src 'self'; " +
	"font-src 'self'; " +
	"img-src 'self' data:; " +
	"connect-src 'self'; " +
	"form-action 'self'; " +
	"base-uri 'none'; " +
	"frame-ancestors 'none'; " +
	"object-src 'none'"

// permissionsPolicy switches off browser features the site never uses, so an
// injected script could not reach for them either.
const permissionsPolicy = "accelerometer=(), camera=(), geolocation=(), " +
	"gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()"

// securityHeaders applies the response headers that belong on every route.
func securityHeaders(next http.Handler) http.Handler {
	// Read once at construction rather than per request.
	forceHSTS := os.Getenv("FORCE_HSTS") == "1"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Frame-Options", "DENY") // for browsers predating frame-ancestors
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		h.Set("Permissions-Policy", permissionsPolicy)

		// HSTS is only meaningful, and only safe, over HTTPS. Behind a
		// TLS-terminating proxy r.TLS is nil, so that deployment opts in
		// explicitly with FORCE_HSTS=1 rather than this trusting a forwarded
		// header any client can set.
		if r.TLS != nil || forceHSTS {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}
