package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"errors"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func parseBase64(k string, d map[string]interface{}) ([]byte, error) {
	v, ok := d[k].(string)
	if !ok {
		return nil, errors.New("key " + k + " not found")
	}
	vv, err := base64.RawURLEncoding.DecodeString(v)
	if err != nil {
		return nil, err
	}
	return vv, nil
}

func parseKey(key map[string]interface{}) (string, interface{}, error) {
	kty, ok := key["kty"].(string)
	if !ok {
		return "", nil, errors.New("kty not found")
	}
	alg, ok := key["alg"].(string)
	if !ok {
		return "", nil, errors.New("alg not found")
	}

	switch kty {
	case "oct":
		var length int
		switch alg {
		case "HS256":
			length = 32
		case "HS384":
			length = 48
		case "HS512":
			length = 64
		default:
			return "", nil, errors.New("unknown alg")
		}
		k, err := parseBase64("k", key)
		if err != nil {
			return "", nil, err
		}
		if len(k) != length {
			return "", nil, errors.New("bad length for key")
		}
		return alg, k, nil
	case "EC":
		if alg != "ES256" {
			return "", nil, errors.New("uknown alg")
		}
		crv, ok := key["crv"].(string)
		if !ok {
			return "", nil, errors.New("crv not found")
		}
		if crv != "P-256" {
			return "", nil, errors.New("unknown crv")
		}
		curve := elliptic.P256()
		xbytes, err := parseBase64("x", key)
		if err != nil {
			return "", nil, err
		}
		var x big.Int
		x.SetBytes(xbytes)
		ybytes, err := parseBase64("y", key)
		if err != nil {
			return "", nil, err
		}
		var y big.Int
		y.SetBytes(ybytes)
		if !curve.IsOnCurve(&x, &y) {
			return "", nil, errors.New("key is not on curve")
		}
		return alg, &ecdsa.PublicKey{
			Curve: curve,
			X:     &x,
			Y:     &y,
		}, nil
	default:
		return "", nil, errors.New("unknown key type")
	}
}

func makeToken(alg string, key interface{}, issuer, location, username, password string) (string, error) {
	now := time.Now()

	m := make(map[string]interface{})
	if issuer != "" {
		m["iss"] = issuer
	}
	if location != "" {
		m["aud"] = location
	}
	if username != "" {
		m["sub"] = username
	}
	m["permissions"] = []string{"present"}
	m["iat"] = now.Add(-time.Second).Unix()
	m["exp"] = now.Add(30 * time.Second).Unix()

	method := jwt.GetSigningMethod(alg)
	if method == nil {
		return "", errors.New("unknown alg")
	}
	token := jwt.NewWithClaims(method, jwt.MapClaims(m))
	return token.SignedString(key)
}
