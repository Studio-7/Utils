package customtype

import (
	"log"
	"math/rand"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	au "github.com/devd-99/Utils/utils"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// Enum for choosing what details to be updated
const (
	FName = iota + 1
	LName
	UName
	Salt
	Passwd
	Email
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// SetProfilePic sets user profile picture
// TODO(1) - Update with image utils to pass only image location
func (user *User) SetProfilePic(img Image) {
	user.ProfilePic = img
}

// CreatePassword operates on a user struct
// It calculates a random salt and hash and stores it in the user struct
func (user *User) CreatePassword(p string) {
	salt := randStringBytes(32)
	user.Salt = salt
	p = p + salt
	hash, err := hashPassword(p)
	if err != nil {
		log.Fatal(err)
	}
	user.Passwd = hash
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash takes in a password that is not hashed and
// compares it with a hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// UpdateDetails takes a user struct and checks it's userid to get the value from the table.
// Empty values are ignored and new values are updated into the object which is then stored into the database.
func (user *User) UpdateDetails(username string, password string, session *r.Session) string {

	conf := ""
	db := os.Getenv("DB")
	var u User
	userTable := os.Getenv("USERTABLE")
	var conf string
	if au.CheckUserExists(username, userTable, session) {
		cur, _ := r.DB(db).Table(userTable).GetAllByIndex("username", username).Run(session)
		_ = cur.One(&u)
		cur.Close()
	}

	if user.UName == "" {
		user.UName = u.UName
	}
	if user.Passwd == "" {
		user.Passwd = u.Passwd
	}
	if user.Email == "" {
		user.Email = u.Email
	}
	if user.FName == "" {
		user.FName = u.FName
	}
	if user.LName == "" {
		user.LName = u.LName
	}

	cur, _ := r.DB(db).Table(userTable).Get(username).Update(user).RunWrite(session)
	if _ != nil {
		log.Fatal(_)
		conf = _.error()
	}
	return conf

}
