package auth

import (
	"testing"
	"watchparty-backend/internal/store"
)

func TestAuthFlow(t *testing.T) {
	memStore := store.NewMemoryStore()
	authService := NewAuthService("test-secret-key", memStore)

	// 1. Request OTP
	code, err := authService.RequestOTP("shubh@example.com")
	if err != nil {
		t.Fatalf("request otp failed: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6-digit code, got %s", code)
	}

	// 2. Verify with valid OTP
	user, token, err := authService.VerifyOTP("shubh@example.com", code, "Shubh", "")
	if err != nil {
		t.Fatalf("verify otp failed: %v", err)
	}
	if user.Name != "Shubh" || token == "" {
		t.Fatalf("invalid user or token: %+v", user)
	}

	// 3. Validate JWT
	claims, err := authService.ValidateToken(token)
	if err != nil {
		t.Fatalf("validate token failed: %v", err)
	}
	if claims.UserID != user.ID || claims.Email != user.Email {
		t.Fatalf("claims mismatch: %+v", claims)
	}

	// 4. Test development master OTP "123456"
	user2, token2, err := authService.VerifyOTP("gf@example.com", "123456", "Girlfriend", "")
	if err != nil {
		t.Fatalf("master dev otp should succeed: %v", err)
	}
	if user2.Name != "Girlfriend" || token2 == "" {
		t.Fatalf("invalid user or token: %+v", user2)
	}
}
