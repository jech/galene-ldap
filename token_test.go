package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

var keys = []string{
	`{"alg":"HS256","k":"ZUtxl5wJ1db-erWkLYAYCxx3d0PdFDL8f903SF8k08M","key_ops":["sign","verify"],"kty":"oct"}`,
	`{"alg":"HS384","k":"_u5991ROm0v4KTcdZVWGWmH3ol3vyYCnCVkNlAnaDXPKWvuUAV3xwH9WJ22jeSqv","key_ops":["sign","verify"],"kty":"oct"}`,
	`{"alg":"HS512","k":"YHtmwudhfEDeEYxiVaHjHGlH3epeKB4CL8mot5XKMTvAuSq0H2bp9ZEYi6s4pkwwYS20WophkdvujORLWaowaA","key_ops":["sign","verify"],"kty":"oct"}`,
	`{"alg":"ES256","crv":"P-256","d":"c1kE6PgP6ik1Ga4KvQ9srvuZ8435UOi4cyzQOc2Dx_g","key_ops":["sign","verify"],"kty":"EC","x":"HyZNurU1941I0u9CFXKzd6g31SbHRHqpPA9dlHb9Plo","y":"neERCfvQ-SIX8D7V-HqcRAZb8Ixyvu0wNgj_URWR8r8"}`,
}

func TestSign(t *testing.T) {
	for i, k := range keys {
		var j map[string]interface{}
		err := json.Unmarshal([]byte(k), &j)
		if err != nil {
			t.Errorf("Couldn't parse JSON for key %v: %v", i, err)
			continue
		}

		alg, key, err := parseKey(j)
		if err != nil {
			t.Errorf("Couldn't parse key %v: %v", i, err)
			continue
		}

		token, err := makeToken(
			alg, key, "issuer", "location", "username", "password",
		)
		if err != nil {
			t.Errorf("Couldn't generate token %v: %v", i, err)
			continue
		}

		_, err = jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			switch kk := key.(type) {
			case []byte:
				return kk, nil
			case *ecdsa.PrivateKey:
				return &kk.PublicKey, nil
			default:
				return nil, errors.New("unexpected key type")
			}
		})
		if err != nil {
			t.Errorf("Couldn't parse key %v: %v", i, err)
		}
	}
}
