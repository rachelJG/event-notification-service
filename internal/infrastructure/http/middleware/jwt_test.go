package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func makeToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

func newJWTRouter(opts JWTOptions) *gin.Engine {
	r := gin.New()
	r.POST("/test", JWTAuth(opts), func(c *gin.Context) {
		sub, _ := c.Get(ContextKeySubject)
		c.JSON(http.StatusOK, gin.H{"sub": sub})
	})
	return r
}

func TestJWTAuthSuccess(t *testing.T) {
	secret := "supersecretkeythatisatleast32bytes!!"
	token := makeToken(t, secret, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})

	r := newJWTRouter(JWTOptions{Secret: secret})
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestJWTAuthSubjectStoredInContext(t *testing.T) {
	secret := "supersecretkeythatisatleast32bytes!!"
	token := makeToken(t, secret, jwt.MapClaims{
		"sub": "user-42",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})

	var capturedSub string
	r := gin.New()
	r.POST("/test", JWTAuth(JWTOptions{Secret: secret}), func(c *gin.Context) {
		val, _ := c.Get(ContextKeySubject)
		capturedSub, _ = val.(string)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(httptest.NewRecorder(), req)

	if capturedSub != "user-42" {
		t.Fatalf("expected subject 'user-42', got %q", capturedSub)
	}
}

func TestJWTAuthMissingHeader(t *testing.T) {
	r := newJWTRouter(JWTOptions{Secret: "supersecretkeythatisatleast32bytes!!"})
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuthExpiredToken(t *testing.T) {
	secret := "supersecretkeythatisatleast32bytes!!"
	token := makeToken(t, secret, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(-1 * time.Minute).Unix(),
	})

	r := newJWTRouter(JWTOptions{Secret: secret})
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestJWTAuthMissingExpClaim(t *testing.T) {
	secret := "supersecretkeythatisatleast32bytes!!"
	// Token without exp claim
	token := makeToken(t, secret, jwt.MapClaims{"sub": "user-1"})

	r := newJWTRouter(JWTOptions{Secret: secret})
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for token without exp, got %d", w.Code)
	}
}

func TestJWTAuthWrongAlgorithm(t *testing.T) {
	// Sign with RS256 (asymmetric) — middleware only accepts HS256
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})
	signed, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	r := newJWTRouter(JWTOptions{Secret: "supersecretkeythatisatleast32bytes!!"})
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for RS256 token, got %d", w.Code)
	}
}

func TestJWTAuthIssuerValidation(t *testing.T) {
	secret := "supersecretkeythatisatleast32bytes!!"

	validToken := makeToken(t, secret, jwt.MapClaims{
		"sub": "user-1",
		"iss": "my-service",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})
	wrongIssuerToken := makeToken(t, secret, jwt.MapClaims{
		"sub": "user-1",
		"iss": "other-service",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})

	opts := JWTOptions{Secret: secret, Issuer: "my-service"}

	for _, tc := range []struct {
		name       string
		token      string
		wantStatus int
	}{
		{"valid issuer", validToken, http.StatusOK},
		{"wrong issuer", wrongIssuerToken, http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newJWTRouter(opts)
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func TestJWTAuthAudienceValidation(t *testing.T) {
	secret := "supersecretkeythatisatleast32bytes!!"

	validToken := makeToken(t, secret, jwt.MapClaims{
		"sub": "user-1",
		"aud": []string{"events-api"},
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})
	wrongAudToken := makeToken(t, secret, jwt.MapClaims{
		"sub": "user-1",
		"aud": []string{"other-api"},
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})

	opts := JWTOptions{Secret: secret, Audience: "events-api"}

	for _, tc := range []struct {
		name       string
		token      string
		wantStatus int
	}{
		{"valid audience", validToken, http.StatusOK},
		{"wrong audience", wrongAudToken, http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newJWTRouter(opts)
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func TestJWTAuthEmptySecret(t *testing.T) {
	r := newJWTRouter(JWTOptions{Secret: ""})
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for empty secret, got %d", w.Code)
	}
}

func TestJWTAuthWithLoggerOnFailure(t *testing.T) {
	secret := "supersecretkeythatisatleast32bytes!!"
	// Expired token triggers the failure-logging branch
	token := makeToken(t, secret, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(-1 * time.Minute).Unix(),
	})

	logger, _ := zap.NewDevelopment()
	r := newJWTRouter(JWTOptions{Secret: secret, Logger: logger})
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuthWithLoggerOnSuccess(t *testing.T) {
	secret := "supersecretkeythatisatleast32bytes!!"
	token := makeToken(t, secret, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})

	logger, _ := zap.NewDevelopment()
	r := newJWTRouter(JWTOptions{Secret: secret, Logger: logger})
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

