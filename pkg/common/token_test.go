package common

import "testing"

func TestTokenSignUnsign(t *testing.T) {
	secret := "abcd"
	value := "email@domain.com"

	signed := Sign(secret, value)
	unsigned, success := Unsign(secret, signed)

	if !success {
		t.Errorf("Failed to unsign")
	}

	if unsigned != value {
		t.Errorf("Values do not match. unsigned=%v value=%v", unsigned, value)
	}
}
