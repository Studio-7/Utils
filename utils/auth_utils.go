package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"

	ct "github.com/cvhariharan/Data-Models/customtype"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

type AuthToken struct {
	Token string
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func computeHMAC(message string) string {
	key := []byte(os.Getenv("HMACKEY"))
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// GenerateJWT takes a username and generates a JWT with HMAC
func GenerateJWT(username string) string {
	salt := randStringBytes(32)
	u64 := b64.URLEncoding.EncodeToString([]byte(username))
	s64 := b64.URLEncoding.EncodeToString([]byte(salt))
	hash := computeHMAC(u64 + "." + s64)
	return u64 + "." + s64 + "." + b64.URLEncoding.EncodeToString([]byte(hash))
}

// ValidateJWT takes in a jew string and returns the username if it is valid
// else it returns an empty string
func ValidateJWT(jwt string) string {
	var username string
	if jwt != "" {
		parts := strings.Split(jwt, ".")
		if len(parts) == 3 {
			u, _ := b64.URLEncoding.DecodeString(parts[0])
			// s, _ := b64.URLEncoding.DecodeString(parts[1])
			h, _ := b64.URLEncoding.DecodeString(parts[2])
			hash := computeHMAC(parts[0] + "." + parts[1])
			if hash == string(h) {
				return string(u)
			}
		}
	}
	return username
}

// CheckUserExists takes in a username and db session and check if the user
// exists in the table
func CheckUserExists(username string, session *r.Session) bool {
	var u interface{}
	db := os.Getenv("DB")
	userTable := os.Getenv("USERTABLE")
	cur, _ := r.DB(db).Table(userTable).GetAllByIndex("username", username).Run(session)
	_ = cur.One(&u)
	// fmt.Println(u)
	if u == nil {
		return false
	}
	return true
}

// UserSignup takes a new user struct, inserts it into table and returns a JWT
// If username exists, returns empty string
func UserSignup(user ct.User, session *r.Session) string {
	var jwt string
	db := os.Getenv("DB")
	userTable := os.Getenv("USERTABLE")
	// Check if username exists
	if !CheckUserExists(user.UName, session) {
		// fmt.Println("No")
		_, err := r.DB(db).Table(userTable).Insert(user).Run(session)
		if err != nil {
			log.Fatal(err)
		}
		jwt = GenerateJWT(user.UName)
	}

	return jwt
}

// UserLogin takes a user struct and checks if it exists in the table
// if yes, then returns a jwt else returns empty string
func UserLogin(user ct.User, session *r.Session) string {
	var jwt string
	var u map[string]interface{}
	db := os.Getenv("DB")
	userTable := os.Getenv("USERTABLE")
	if CheckUserExists(user.UName, session) {
		cur, _ := r.DB(db).Table(userTable).GetAllByIndex("username", user.UName).Run(session)
		_ = cur.One(&u)

		fmt.Println(u["password"])
		fmt.Println(user.Passwd)
		if user.Passwd == u["password"] {
			// User authenticated
			jwt = GenerateJWT(user.UName)
		}
	}
	return jwt
}

// AuthMiddleware takes in a JWT and http.handler and
// returns the handler if JWT is valid
func AuthMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]interface{}
		request, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.Unmarshal([]byte(request), &data)
		recvToken, recvOk := data["token"]
		username, usernameOk := data["username"]
		if usernameOk && recvOk && ValidateJWT(recvToken.(string)) == username.(string) {
			handler.ServeHTTP(w, r)
		} else {
			fmt.Fprint(w, "Not Authorized")
		}
	})
}
