package main

import (
	"fmt"
	"log"

	"github.com/cvhariharan/Data-Models/customtype"
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

	user := customtype.User{FName: "a", LName: "b"}
	user.CreatePassword("testpass")
	user.UpdateDetails("testuser2", customtype.UName)
	fmt.Println(utils.UserSignup(user, session))
	fmt.Println("Login")
	fmt.Println(utils.UserLogin(user, session))
}
