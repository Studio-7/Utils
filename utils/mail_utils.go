package utils

import (
	"fmt"
	"log"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// SendMail takes in toEmail, username, subject and message and send an email
func SendMail(toEmail string, username string, subject string, message string) {
	apikey := os.Getenv("SENDGRIDAPI")
	from := mail.NewEmail("Postcard", "postcard@postcard.com")
	to := mail.NewEmail(username, toEmail)
	res := mail.NewSingleEmail(from, subject, to, message, "")
	client := sendgrid.NewSendClient(apikey)
	response, err := client.Send(res)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}

// CheckEmailExists checks the user table and returns true if email already used, else false
func CheckEmailExists(email string, session *r.Session) bool {
	var u interface{}
	db := os.Getenv("DB")
	userTable := os.Getenv("USERTABLE")
	cur, _ := r.DB(db).Table(userTable).GetAllByIndex("email", email).Run(session)
	_ = cur.One(&u)
	// fmt.Println(u)
	if u == nil {
		return false
	}
	return true
}
