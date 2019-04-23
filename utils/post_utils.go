package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	ct "github.com/cvhariharan/Utils/customtype"
	"github.com/xiam/exif"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

type ipfsResp struct {
	Name string `json:"Name"`
	Hash string `json:"Hash"`
	Size string `json:"Size"`
}

// uploadToIPFS takes image location from temp-dir and
// uploads it to IPFS and returns the cid
func uploadToIPFS(imgloc string) string {
	var ipfsresp ipfsResp
	url := os.Getenv("IPFS")
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	defer writer.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(imgloc))
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Open(imgloc)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	if _, err = io.Copy(part, file); err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post(url, writer.FormDataContentType(), buf)
	if err != nil {
		fmt.Println(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	json.Unmarshal(body, &ipfsresp)
	return ipfsresp.Hash
}

func getImage(imgloc string, session *r.Session) ct.Image {
	var img ct.Image
	db := os.Getenv("DB")
	imgTable := os.Getenv("IMGTABLE")
	cid := uploadToIPFS(imgloc)
	fmt.Println(cid)
	img = ct.Image{
		Link: cid,
	}
	img.UploadedOn = time.Now()
	if cid != "" {
		data, _ := exif.Read(imgloc)
		if data != nil {
			img.CreatedOn = data.Tags["Date and Time"]
			img.Manufacturer = data.Tags["Manufacturer"]
			img.Model = data.Tags["Model"]
		}
	}
	// Add to DB
	r.DB(db).Table(imgTable).Insert(img).Run(session)
	return img
}

// CheckPostExists takes in a post id and returns true
// if it exists
func CheckPostExists(postId string, session *r.Session) bool {
	var post interface{}
	db := os.Getenv("DB")
	postTable := os.Getenv("POSTTABLE")
	cur, _ := r.DB(db).Table(postTable).Get(postId).Run(session)
	cur.One(&post)
	fmt.Println(post)
	if post != nil {
		return true
	}
	return false
}

// CheckTravelCapsuleExists takes in a travelcapsule id and returns the username
// of the creator if the travelcapsule exists.
func CheckTravelCapsuleExists(travelcapsule string, session *r.Session) string {
	if travelcapsule == "" {
		return ""
	}
	var t ct.TravelCapsule
	db := os.Getenv("DB")
	tcTable := os.Getenv("TCTABLE")
	cur, _ := r.DB(db).Table(tcTable).Get(travelcapsule).Run(session)
	_ = cur.One(&t)
	cur.Close()
	return t.CreatedBy
}

// addImageToPost runs concurrently and inserts the image, if any to
// the corresponding post after uploading it to IPFS.
func addImageToPost(imgloc, travelcapsule string, post ct.Post, session *r.Session) {
	fmt.Println("addImage")
	db := os.Getenv("DB")
	postTable := os.Getenv("POSTTABLE")
	tcTable := os.Getenv("TCTABLE")
	if imgloc != "" {
		img := getImage(imgloc, session)
		post.PostBody.Img = img
	}

	// Insert into post table
	insertedPost, err := r.DB(db).Table(postTable).Insert(post).RunWrite(session)
	if err != nil {
		fmt.Println(err)
	}

	// Store the id in the post to be inserted in TC
	id := insertedPost.GeneratedKeys[0]

	r.DB(db).Table(tcTable).Get(travelcapsule).
			Update(map[string]interface{}{"posts": r.Row.Field("posts").
			Append(id),
			"updated_on": time.Now(),}).
			RunWrite(session)
	
}

// CreatePost takes in a travelcapsule id and adds the post to it. 
// On success it returns the tc id else it returns empty string
// Created to prevent direct handling of data-model structs in the handlers
func CreatePost(travelcapsule, title, message, imgloc string, hashtags []string, username string, session *r.Session) string {
	capsule := travelcapsule

	if travelcapsule != "" {
		var body ct.Body
		body = ct.Body{
			Message: message,
		}
		
		post := ct.Post{
			Title:     title,
			CreatedOn: time.Now(),
			CreatedBy: username,
			PostBody:  body,
			Hashtags:  hashtags,
			Likes: 0,
		}
		creator := CheckTravelCapsuleExists(travelcapsule, session)
		fmt.Println("Creator: " + creator + " User: " + username)
		if creator == username {
			go addImageToPost(imgloc, travelcapsule, post, session)
			fmt.Println("Added image")
			return capsule
		}
	}
	return ""
}

// CreateTC takes in a title and username and creates a new TC
func CreateTC(title, username string, session *r.Session) string {
	var capsule string
	db := os.Getenv("DB")
	tcTable := os.Getenv("TCTABLE")
	tc := ct.TravelCapsule{
		CreatedOn: time.Now(),
		CreatedBy: username,
		UpdatedOn: time.Now(),
		Title: title,
	}
	insertedTerm, err := r.DB(db).Table(tcTable).Insert(tc).RunWrite(session)
	if err != nil {
		fmt.Println(err)
	}
	capsule = insertedTerm.GeneratedKeys[0]
	return capsule
}

// GetTC takes in a postId and returns a slice of 
// all the TCs containing that post
func GetTC(postId string, session *r.Session) []string {
	var tcs []string
	var tc ct.TravelCapsule
	db := os.Getenv("DB")
	tcTable := os.Getenv("TCTABLE")
	cur, _ := r.DB(db).Table(tcTable).GetAllByIndex("posts", postId).Run(session)

	for cur.Next(&tc) {
		tcs = append(tcs, tc.Id)
	}
	return tcs
}


// GetSimplifiedPost returns a simpler version of a post
func GetPost(postId string, simplified bool, session *r.Session) interface{} {
	var simplePost ct.SimplePost
	var post ct.Post
	db := os.Getenv("DB")
	postTable := os.Getenv("POSTTABLE")
	cur, _ := r.DB(db).Table(postTable).Get(postId).Run(session)
	cur.One(&post)
	simplePost = ct.SimplePost{
		Id: post.Id,
		Title: post.Title,
		CreatedOn: post.CreatedOn,
		Message: post.PostBody.Message,
		Image: "https://cloudflare-ipfs.com/ipfs/" + post.PostBody.Img.Link,
	}
	fmt.Println(simplePost)
	if simplified {
		return simplePost
	}
	return post
}
