package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	b64 "encoding/base64"
	"fmt"
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

// resetJWT takes a username and deletes that user's token
// to be called after login
func resetJWT(username string, session *r.Session) {
	db := os.Getenv("DB")
	table := os.Getenv("TOKENTABLE")
	r.DB(db).Table(table).GetAllByIndex("username", username).Delete().Run(session)
}

// GenerateJWT takes a username and generates a JWT with HMAC
func GenerateJWT(username string, session *r.Session) string {
	var jwt string
	db := os.Getenv("DB")
	tokenTable := os.Getenv("TOKENTABLE")
	salt := randStringBytes(32)
	u64 := b64.URLEncoding.EncodeToString([]byte(username))
	s64 := b64.URLEncoding.EncodeToString([]byte(salt))
	hash := computeHMAC(u64 + "." + s64)
	h := u64 + "." + s64 + "." + b64.URLEncoding.EncodeToString([]byte(hash))
	// Write to token table
	if !CheckUserExists(username, tokenTable, session) {
		auth := AuthToken{username, h}
		// fmt.Println(auth)
		r.DB(db).Table(tokenTable).Insert(auth).Run(session)
		jwt = h
	}

	return jwt
}

// ValidateJWT takes in a jew string and returns the username if it is valid
// else it returns an empty string
func ValidateJWT(jwt string, session *r.Session) string {
	var auth interface{}
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
					cur, _ := r.DB(db).Table(tokenTable).GetAllByIndex("token", jwt).Run(session)
					cur.One(&auth)
					if auth != nil {
						// Token exists in table, ie valid token
						// Delete the currently used token from tokentable
						r.DB(db).Table(tokenTable).GetAllByIndex("username", username).Delete().Run(session)
						return username
					}

				}
			}
		}
	}
	return ""
}

// GetUser returns the user object by taking the username as input
func GetUser(username string, session *r.Session) ct.User {
	var u ct.User
	// var user ct.User
	db := os.Getenv("DB")
	table := os.Getenv("USERTABLE")
	// userTable := os.Getenv("USERTABLE")
	cur, _ := r.DB(db).Table(table).GetAllByIndex("username", username).Run(session)
	_ = cur.One(&u)
	cur.Close()
	// fmt.Println(u)
	// mapstructure.Decode(u, &user)
	return u
}

// CheckUserExists takes in a username, table and db session and check if the user
// exists in the given table
func CheckUserExists(username string, table string, session *r.Session) bool {
	var u interface{}
	db := os.Getenv("DB")
	// userTable := os.Getenv("USERTABLE")
	cur, _ := r.DB(db).Table(table).GetAllByIndex("username", username).Run(session)
	_ = cur.One(&u)
	cur.Close()
	// fmt.Println(u)
	if u == nil {
		// fmt.Println("NO")
		return false
	}
	// fmt.Println("YES")
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
		// SendMail(user.Email, user.UName, "OTP", randStringBytes(6))
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
			resetJWT(username, session)
			jwt = GenerateJWT(username, session)
		}
	}
	return jwt
}

// AuthMiddleware takes in a JWT and http.handler and
// returns the handler if JWT is valid
func AuthMiddleware(handler http.HandlerFunc, session *r.Session) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		var username string
		fmt.Println(r.Header.Get("Content-Type"))
		if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			// fmt.Println("Multipart")
			r.ParseForm()
			token = r.Form.Get("token")
			username = r.Form.Get("username")
		} else if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			// fmt.Println("Multipart")
			r.ParseMultipartForm(32 << 20)
			token = r.FormValue("token")
			username = r.FormValue("username")
			fmt.Println(token + ":" + username)
		}
		if token != "" && username != "" && ValidateJWT(token, session) == username {
			// fmt.Println("Authorized")
			handler.ServeHTTP(w, r)
		} else {
			fmt.Fprint(w, `{ "error" : "Not Authorized"}`)
		}
	})
}

// UpdateUserDetails taken in the username, and a new user object containing all new user data
// it passes the data to UpdateDetails in user_utils, and then passes the confirmation of the
// updation to the handler
func UpdateUserDetails(username string, user ct.user, session *r.session) {

	conf := user.UpdateDetails(username, *r.session)
	return conf

}
