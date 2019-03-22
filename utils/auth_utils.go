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

	ct "github.com/cvhariharan/Utils/customtype"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

type AuthToken struct {
	Username string `rethinkdb:"username"`
	Token    string `rethinkdb:"token"`
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
func GenerateJWT(username string, session *r.Session) string {
	db := os.Getenv("DB")
	tokenTable := os.Getenv("TOKENTABLE")
	salt := randStringBytes(32)
	u64 := b64.URLEncoding.EncodeToString([]byte(username))
	s64 := b64.URLEncoding.EncodeToString([]byte(salt))
	hash := computeHMAC(u64 + "." + s64)
	jwt := u64 + "." + s64 + "." + b64.URLEncoding.EncodeToString([]byte(hash))
	// Write to token table
	if !CheckUserExists(username, tokenTable, session) {
		auth := AuthToken{username, jwt}
		fmt.Println(auth)
		r.DB(db).Table(tokenTable).Insert(auth).Run(session)
	}

	return jwt
}

// ValidateJWT takes in a jew string and returns the username if it is valid
// else it returns an empty string
func ValidateJWT(jwt string, session *r.Session) string {
	tokenTable := os.Getenv("TOKENTABLE")
	db := os.Getenv("DB")
	var username string
	if jwt != "" {
		parts := strings.Split(jwt, ".")
		if len(parts) == 3 {
			u, _ := b64.URLEncoding.DecodeString(parts[0])
			// s, _ := b64.URLEncoding.DecodeString(parts[1])
			h, _ := b64.URLEncoding.DecodeString(parts[2])
			hash := computeHMAC(parts[0] + "." + parts[1])
			if hash == string(h) {
				if CheckUserExists(string(u), tokenTable, session) {
					username = string(u)
					// Delete the currently used token from tokentable
					r.DB(db).Table(tokenTable).GetAllByIndex("username", username).Delete().Run(session)
				}
			}
		}
	}
	return username
}

// func GetUser(username string, session *r.Session) ct.User {
// 	var u ct.User
// 	// var user ct.User
// 	db := os.Getenv("DB")
// 	table := os.Getenv("USERTABLE")
// 	// userTable := os.Getenv("USERTABLE")
// 	cur, _ := r.DB(db).Table(table).GetAllByIndex("username", username).Run(session)
// 	_ = cur.One(&u)
// 	// fmt.Println(u)
// 	// mapstructure.Decode(u, &user)
// 	return u
// }

// CheckUserExists takes in a username, table and db session and check if the user
// exists in the given table
func CheckUserExists(username string, table string, session *r.Session) bool {
	var u interface{}
	db := os.Getenv("DB")
	// userTable := os.Getenv("USERTABLE")
	cur, _ := r.DB(db).Table(table).GetAllByIndex("username", username).Run(session)
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
	// Check if username or email exists
	// fmt.Println(user.UName)
	if !CheckUserExists(user.UName, userTable, session) && user.UName != "" && !CheckEmailExists(user.Email, session) {
		// fmt.Println("No")
		_, err := r.DB(db).Table(userTable).Insert(user).Run(session)
		if err != nil {
			log.Fatal(err)
		}
		jwt = GenerateJWT(user.UName, session)
	}

	return jwt
}

// UserLogin takes a user struct and checks if it exists in the table
// if yes, then returns a jwt else returns empty string
func UserLogin(username string, password string, session *r.Session) string {
	var jwt string
	var u map[string]interface{}
	db := os.Getenv("DB")
	userTable := os.Getenv("USERTABLE")
	if CheckUserExists(username, userTable, session) {
		cur, _ := r.DB(db).Table(userTable).GetAllByIndex("username", username).Run(session)
		_ = cur.One(&u)
		salt := u["salt"].(string)
		// fmt.Println(salt)
		// user := ct.User{UName: username}
		// hash := ct.HashPasswordWithSalt(password, salt)
		// fmt.Println(hash)
		// fmt.Println(u["password"])
		if ct.CheckPasswordHash(password+salt, u["password"].(string)) {
			// User authenticated
			jwt = GenerateJWT(username, session)
		}
	}
	return jwt
}

// AuthMiddleware takes in a JWT and http.handler and
// returns the handler if JWT is valid
func AuthMiddleware(handler http.HandlerFunc, session *r.Session) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]interface{}
		request, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.Unmarshal([]byte(request), &data)
		recvToken, recvOk := data["token"]
		username, usernameOk := data["username"]
		if usernameOk && recvOk && ValidateJWT(recvToken.(string), session) == username.(string) {
			handler.ServeHTTP(w, r)
		} else {
			fmt.Fprint(w, "Not Authorized")
		}
	})
}
