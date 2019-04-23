package customtype

import (
	"log"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
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

// UpdateDetails operates on user struct to update the user info
// To update the salt, the current password has to be passed
func (user *User) UpdateDetails(newData string, field int) {
	if newData != "" {
		switch field {
		case FName:
			user.FName = newData
		case LName:
			user.LName = newData
		case UName:
			user.UName = newData
		case Passwd:
			user.CreatePassword(newData)
		case Salt:
			user.CreatePassword(newData)
		case Email:
			user.Email = newData
		}
	}
}
