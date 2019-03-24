package main

import (
	"fmt"
	"log"

	"github.com/cvhariharan/Utils/customtype"
	"github.com/cvhariharan/Utils/utils"
	"github.com/joho/godotenv"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func main() {
	e := godotenv.Load()
	if e != nil {
		log.Fatal(e)
	}

	url := "localhost:28015"
	session, err := r.Connect(r.ConnectOpts{
		Address: url, // endpoint without http
	})
	if err != nil {
		log.Fatalln(err)
	}

	user := customtype.User{FName: "yako", LName: "makaza"}
	user.CreatePassword("testpass")
	user.UpdateDetails("3r4", customtype.UName)
	user.UpdateDetails("zeptonews@gmail.com", customtype.Email)
	fmt.Println(utils.UserSignup(user, session))
	// user := utils.GetUser("johnwick1", session)
	// fmt.Println(user)
	// fmt.Println(utils.ValidateJWT("eWFrb21ha2F6YTE=.dXFWVU9FYVl2a0hQd2FHYVVjV0lzU1pYQXdWd3FUTGU=.TSt6NjlDVUpRb0tHYUJ5TDg0TTh0YkpJcVZCMTNaQ3NwMWR0K3hzVmYvST0=", session))
	// fmt.Println("Login")
	// fmt.Println(utils.UserLogin(user, session))
}
