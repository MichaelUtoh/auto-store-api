package paystack

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifyWebhookSignature(t *testing.T) {
	secret := "test_secret"
	body := []byte(`{"event":"charge.success"}`)
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	assert.True(t, VerifyWebhookSignature(secret, body, sig))
	assert.False(t, VerifyWebhookSignature(secret, body, "bad"))
	assert.False(t, VerifyWebhookSignature("", body, sig))
}
