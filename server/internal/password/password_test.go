package password

import (
	"strings"
	"testing"
)

func TestHashVerify(t *testing.T) {
	h := Hash("Admin123!")
	if !strings.HasPrefix(h, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected parameters: %s", h)
	}
	if ok, err := Verify("Admin123!", h); err != nil || !ok {
		t.Fatalf("correct password rejected: %v %v", ok, err)
	}
	if ok, err := Verify("admin123!", h); err != nil || ok {
		t.Fatalf("wrong password accepted: %v %v", ok, err)
	}
	if h2 := Hash("Admin123!"); h2 == h {
		t.Fatal("salt not random")
	}
}

func TestVerifyMalformed(t *testing.T) {
	for _, bad := range []string{"", "plain", "$argon2i$v=19$m=1,t=1,p=1$c2FsdA$a2V5", "$argon2id$v=19$m=x$c2FsdA$a2V5"} {
		if _, err := Verify("x", bad); err == nil {
			t.Errorf("Verify accepted malformed hash %q", bad)
		}
	}
}
