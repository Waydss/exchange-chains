package security

import "testing"

func TestPasswordHasher(t *testing.T) {
	hasher := NewPasswordHasher()

	hash, err := hasher.Hash("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "password123" {
		t.Fatal("password hash contains the source password")
	}

	matches, err := hasher.Matches(hash, "password123")
	if err != nil {
		t.Fatalf("compare password: %v", err)
	}
	if !matches {
		t.Fatal("correct password does not match")
	}

	matches, err = hasher.Matches(hash, "wrong-password")
	if err != nil {
		t.Fatalf("compare wrong password: %v", err)
	}
	if matches {
		t.Fatal("wrong password matches")
	}
}
