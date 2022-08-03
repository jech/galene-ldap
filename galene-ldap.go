package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/jech/cert"
)

// jsonSet is an array that unmarshals as a hashtable.
type jsonSet map[string]bool

func (s *jsonSet) UnmarshalJSON(b []byte) error {
	var a []string
	err := json.Unmarshal(b, &a)
	if err != nil {
		return err
	}
	*s = make(map[string]bool, len(a))
	for _, v := range a {
		(*s)[v] = true
	}
	return nil
}

func (s jsonSet) MarshalJSON() ([]byte, error) {
	a := make([]string, 0, len(s))
	for v, vv := range s {
		if vv {
			a = append(a, v)
		}
	}
	return json.Marshal(a)
}

type configuration struct {
	Groups                 jsonSet                `json:"groups"`
	PasswordFallback       bool                   `json:"passwordFallback"`
	HttpAddress            string                 `json:"httpAddress"`
	Insecure               bool                   `json:"insecure"`
	Key                    map[string]interface{} `json:"key"`
	LdapServer             string                 `json:"ldapServer"`
	LdapBase               string                 `json:"ldapBase"`
	LdapAuthDN             string                 `json:"ldapAuthDN"`
	LdapAuthPassword       string                 `json:"ldapAuthPassword"`
	LdapClientSideValidate bool                   `json:"ldapClientSideValidate"`
}

var debug bool
var config configuration
var signingKey interface{}
var signingKeyAlg string
var verifyCh chan verifyReq

func main() {
	var dataDir string
	flag.StringVar(&dataDir, "data", ".", "data `directory`")
	flag.BoolVar(&debug, "debug", false, "enable debugging")
	flag.Parse()

	configFile := filepath.Join(dataDir, "galene-ldap.json")

	f, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("Open(%v): %v", configFile, err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatalf("Read(%v): %v", configFile, err)
	}

	signingKeyAlg, signingKey, err = parseKey(config.Key)
	if err != nil {
		log.Fatalf("Parse key: %v", err)
	}

	if config.HttpAddress == "" {
		config.HttpAddress = ":8443"
	}

	verifyCh = make(chan verifyReq, 16)
	go verifier(verifyCh)

	http.HandleFunc("/", httpHandler)

	server := &http.Server{
		Addr:              config.HttpAddress,
		ReadHeaderTimeout: 60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if !config.Insecure {
		certificate := cert.New(
			filepath.Join(dataDir, "cert.pem"),
			filepath.Join(dataDir, "key.pem"),
		)
		server.TLSConfig = &tls.Config{
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return certificate.Get()
			},
		}

		err = server.ListenAndServeTLS("", "")
	} else {
		err = server.ListenAndServe()
	}
	log.Fatal(err)
}

func debugf(format string, v ...interface{}) {
	if debug {
		log.Printf(format, v...)
	}
}

type galeneRequest struct {
	Username string `json:"username"`
	Location string `json:"location"`
	Password string `json:"password"`
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers",
			"Authorization, Content-Type",
		)
		return
	}

	if r.Method != "POST" {
		w.Header().Set("Allow", "OPTIONS, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fallback := func() {
		if config.PasswordFallback {
			w.WriteHeader(http.StatusNoContent)
		} else {
			http.Error(w, "Not authorised", http.StatusUnauthorized)
		}
	}

	decoder := json.NewDecoder(r.Body)
	var req galeneRequest
	err := decoder.Decode(&req)
	if err != nil {
		log.Printf("Decode(request): %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Location == "" || req.Password == "" {
		log.Print("Missing field in request.")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	loc, err := url.Parse(req.Location)
	if err != nil {
		log.Printf("Parse(request.location): %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	p := loc.Path
	if !strings.HasPrefix(p, "/group/") {
		debugf("Path doesn't start with /group/")
		fallback()
		return
	}
	group := strings.TrimSuffix(strings.TrimPrefix(p, "/group/"), "/")
	if !config.Groups[group] {
		debugf("Group not found")
		fallback()
		return
	}

	found, valid, err := verify(req.Username, req.Password)
	if err != nil {
		log.Printf("Verify: %v", err)
		http.Error(w, "Internal server error",
			http.StatusInternalServerError)
		return
	}
	debugf("Verify: found=%v, valid=%v", found, valid)
	if !found {
		fallback()
		return
	}

	if !valid {
		http.Error(w, "Not authorised", http.StatusUnauthorized)
		return
	}

	token, err := makeToken(
		signingKeyAlg, signingKey, "", req.Location, req.Username, req.Password,
	)
	if err != nil {
		log.Printf("makeToken: %v", err)
		http.Error(w, "Not authorised", http.StatusUnauthorized)
		return
	}

	w.Header().Set("content-type", "application/jwt")
	w.Header().Set("cache-control", "no-store")
	io.WriteString(w, token)
}

type verifyResp struct {
	found, valid bool
	error        error
}

type verifyReq struct {
	user, password string
	ch             chan verifyResp
}

func verifier(ch <-chan verifyReq) {
	var conn *ldap.Conn
	var err error
	for {
		req, ok := <-ch
		if !ok {
			return
		}
		if conn == nil {
			conn, err = ldapConnect(
				config.LdapServer,
				config.LdapAuthDN,
				config.LdapAuthPassword,
			)
			if err != nil {
				conn = nil
				req.ch <- verifyResp{error: err}
				close(req.ch)
				continue
			}
		}
		found, valid, err :=
			ldapVerify(
				conn, config.LdapClientSideValidate,
				config.LdapAuthDN, config.LdapAuthPassword,
				req.user, req.password)
		if err != nil {
			conn.Close()
			conn = nil
			req.ch <- verifyResp{error: err}
			close(req.ch)
			continue
		}
		req.ch <- verifyResp{found: found, valid: valid}
		close(req.ch)
	}
}

func verify(user, password string) (bool, bool, error) {
	ch := make(chan verifyResp, 1)
	verifyCh <- verifyReq{user: user, password: password, ch: ch}
	resp := <-ch
	return resp.found, resp.valid, resp.error
}
