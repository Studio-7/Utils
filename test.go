package main

import (
	"fmt"
	"log"

	ct "github.com/cvhariharan/Utils/customtype"
	// "github.com/cvhariharan/Utils/utils"
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

	var post ct.Post 
	cur, _ := r.DB("Postcard").Table("post").Get("e3af9be9-b5e5-4122-ac25-f232340a376f").Run(session)
	_ = cur.One(&post)
	cur.Close()
	fmt.Println(post)
}
