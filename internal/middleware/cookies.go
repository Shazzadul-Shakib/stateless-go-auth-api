package middleware

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Shazzadul-Shakib/go-auth-api/internal/apperr"
	"github.com/Shazzadul-Shakib/go-auth-api/internal/utils"
	"github.com/gin-gonic/gin"
)

const (
	AccessCookie  = "access_token"
	RefreshCookie = "refresh_token"

	// the refresh cookie is only sent to the auth endpoints, never to normal API calls.
	refreshCookiePath = "/api/v1/auth"
)

// SetAuthCookies writes both tokens as HttpOnly cookies.
// Env: COOKIE_SECURE (default true), COOKIE_DOMAIN (default host-only), COOKIE_SAMESITE (lax|strict|none, default lax).
func SetAuthCookies(c *gin.Context, accessToken, refreshToken string) error {
	accessTTL, err := utils.AccessTTL()
	if err != nil {
		return err
	}
	refreshTTL, err := utils.RefreshTTL()
	if err != nil {
		return err
	}
	http.SetCookie(c.Writer, authCookie(AccessCookie, accessToken, "/", accessTTL, false))
	http.SetCookie(c.Writer, authCookie(RefreshCookie, refreshToken, refreshCookiePath, refreshTTL, true))
	return nil
}

// ClearAuthCookies expires both cookies in the browser.
func ClearAuthCookies(c *gin.Context) {
	http.SetCookie(c.Writer, authCookie(AccessCookie, "", "/", -time.Second, false))
	http.SetCookie(c.Writer, authCookie(RefreshCookie, "", refreshCookiePath, -time.Second, true))
}

func authCookie(name, value, path string, ttl time.Duration, strict bool) *http.Cookie {
	sameSite := sameSiteFromEnv()
	if strict && sameSite == http.SameSiteLaxMode {
		sameSite = http.SameSiteStrictMode
	}
	ck := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   os.Getenv("COOKIE_DOMAIN"),
		HttpOnly: true,
		Secure:   os.Getenv("COOKIE_SECURE") != "false",
		SameSite: sameSite,
	}
	if ttl < 0 {
		ck.MaxAge = -1
	} else {
		ck.MaxAge = int(ttl.Seconds())
	}
	return ck
}

func sameSiteFromEnv() http.SameSite {
	switch strings.ToLower(os.Getenv("COOKIE_SAMESITE")) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

var ErrCSRF = apperr.New(http.StatusForbidden, "csrf_rejected", "missing X-Requested-With header")

// CSRF requires the X-Requested-With header on state-changing requests that rely on
// cookies. Browsers can't add it cross-origin without a CORS preflight, which only
// allowed origins pass. Requests using a Bearer header (mobile/API clients) are exempt.
func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		if c.GetHeader("Authorization") != "" || c.GetHeader("X-Requested-With") != "" {
			c.Next()
			return
		}
		_, errA := c.Cookie(AccessCookie)
		_, errR := c.Cookie(RefreshCookie)
		if errA != nil && errR != nil {
			c.Next()
			return
		}
		apperr.Respond(c, ErrCSRF)
	}
}
