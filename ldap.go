package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/GehirnInc/crypt"
	_ "github.com/GehirnInc/crypt/md5_crypt"
	_ "github.com/GehirnInc/crypt/sha256_crypt"
	_ "github.com/GehirnInc/crypt/sha512_crypt"
	"github.com/go-ldap/ldap/v3"
)

func ldapConnect(server, authDN, authPW string) (*ldap.Conn, error) {
	conn, err := ldap.DialURL(server)
	if err != nil {
		return nil, err
	}

	conn.SetTimeout(30 * time.Second)

	if authDN != "" {
		err = conn.Bind(authDN, authPW)
		if err != nil {
			conn.Close()
			return nil, err
		}
	}
	return conn, nil
}

func ldapVerify(conn *ldap.Conn, clientside bool, authDN, authPW, user, objectclass, password string) (bool, bool, error) {
	attrs := []string{"dn"}
	if clientside {
		attrs = append(attrs, "userPassword")
	}
	sr, err := conn.Search(ldap.NewSearchRequest(
		config.LdapBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		fmt.Sprintf("(&(objectClass=%s)(uid=%s))",
			ldap.EscapeFilter(objectclass),
			ldap.EscapeFilter(user)),
		attrs,
		nil,
	))

	if err != nil {
		return false, false, err
	}

	if len(sr.Entries) != 1 {
		return false, false, nil
	}

	dn := sr.Entries[0].DN
	valid := false

	if !clientside {
		err = conn.Bind(dn, password)
		if err == nil {
			valid = true
		} else if ldap.IsErrorWithCode(
			err, ldap.LDAPResultInvalidCredentials,
		) {
			valid = false
		} else {
			return false, false, err
		}

		if authDN != "" {
			err = conn.Bind(config.LdapAuthDN, config.LdapAuthPassword)
		} else {
			err = conn.UnauthenticatedBind("")
		}
		if err != nil {
			return false, false, err
		}
	} else {
		pw := sr.Entries[0].GetAttributeValue("userPassword")
		if !strings.HasPrefix(pw, "{CRYPT}$") {
			return false, false,
				errors.New("unsupported password format")
		}
		hashed := strings.TrimPrefix(pw, "{CRYPT}")
		if !crypt.IsHashSupported(hashed) {
			return false, false,
				errors.New("unsupported password format")
		}
		crypter := crypt.NewFromHash(hashed)
		err = crypter.Verify(hashed, []byte(password))
		valid = err == nil
	}

	return true, valid, nil
}
