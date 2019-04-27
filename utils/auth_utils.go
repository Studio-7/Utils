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
	"time"
	"golang.org/x/crypto/bcrypt"
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

// hashString hashes a string using bcrypt
func hashString(text string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(text), 14)
	return string(bytes), err
}

// checkHash takes in a password that is not hashed and
// compares it with a hash
func checkHash(text, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(text))
	return err == nil
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
	jwt = h
	// Write to token table
	// if !CheckUserExists(username, tokenTable, session) {
	if CheckUserExists(username, tokenTable, session) {
		// Delete the previous token
		r.DB(db).Table(tokenTable).GetAllByIndex("username", username).Delete().Run(session)
	}
	hash, err := hashString(h)
	if err != nil {
		fmt.Println(err)
	}
	auth := AuthToken{username, hash}
	// fmt.Println(auth)
	r.DB(db).Table(tokenTable).Insert(auth).Run(session)


	return jwt
}

// ValidateJWT takes in a jwt string and returns the username if it is valid
// else it returns an empty string
func ValidateJWT(jwt string, session *r.Session) string {
	var auth AuthToken
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
					// hashedJwt, _ := hashString(jwt)
					fmt.Println(jwt)
					// fmt.Println(hashedJwt)
					cur, _ := r.DB(db).Table(tokenTable).GetAllByIndex("username", username).Run(session)
					cur.One(&auth)
					if auth.Token != "" {
						fmt.Println(auth.Token)
						if checkHash(jwt, auth.Token) {
							// Token exists in table, ie valid token
							// Delete the currently used token from tokentable
							r.DB(db).Table(tokenTable).GetAllByIndex("username", username).Delete().Run(session)
							return username
						}
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

// GetProfile gets the profile details for the given username
func GetProfile(username string, session *r.Session) ct.Profile {
	var profile ct.Profile
	db := os.Getenv("DB")
	userTable := os.Getenv("USERTABLE")

	if CheckUserExists(username, userTable, session) {
		followers := GetFollowers(username, session)
		following := GetFollowees(username, session)
		followersCount := len(followers)
		followingCount := len(following)
		images := GetImages(username, session)
		tcs := GetAllTCs(username, session)

		cur, _ := r.DB(db).Table(userTable).GetAllByIndex("username", username).Run(session)
		cur.One(&profile)
		fmt.Println(profile)
		profile.Followers = followers
		profile.Following = following
		profile.FollowersCount = followersCount
		profile.FollowingCount = followingCount
		profile.TravelCapsules = tcs
		profile.Images = images
	}

	return profile

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

func addProfileImage(username, imgloc string, session *r.Session) {
	db := os.Getenv("DB")
	userTable := os.Getenv("USERTABLE")
	cid := uploadToIPFS(imgloc)
	imgLink := "https://ipfs.io/ipfs/" + cid
	img := ct.Image{
		Link: imgLink,
		CreatedBy: username,
		UploadedOn: time.Now(),
	}

	r.DB(db).Table(userTable).GetAllByIndex("username", username).Update(map[string]interface{}{
		"profile_pic": img,
	}).Run(session)
}

func UpdateProfilePic(username, imgloc string, session *r.Session) bool {
	if imgloc != "" {
		go addProfileImage(username, imgloc, session)
		return true
	}
	return false
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