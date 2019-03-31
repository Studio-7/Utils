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

// CreatePost take params, creates a post object and inserts it into the post table
// created to prevent direct handling of data-model structs in the handlers
func CreatePost(travelcapsule, title, message, imgloc string, hashtags []string, username string, session *r.Session) string {
	capsule := travelcapsule
	db := os.Getenv("DB")
	postTable := os.Getenv("POSTTABLE")
	tcTable := os.Getenv("TCTABLE")

	var img ct.Image
	var body ct.Body

	if imgloc != "" {
		img = getImage(imgloc, session)
		body = ct.Body{
			Message: message,
			Img:     img,
		}
	} else {
		body = ct.Body{
			Message: message,
		}
	}
	
	post := ct.Post{
		Title:     title,
		CreatedOn: time.Now(),
		CreatedBy: username,
		PostBody:  body,
		Hashtags:  hashtags,
		Likes: 0,
	}
	// Attach post to travelcapsule if travelcapsule is not empty
	// else create a new travelcapsule
	if travelcapsule == "" {
		// var t ct.TravelCapsule
		tc := ct.TravelCapsule{
			CreatedOn: time.Now(),
			CreatedBy: username,
			// Hashtags: hashtags,
		}
		// Insert post into table
		p, _ := r.DB(db).Table(postTable).Insert(post).RunWrite(session)
		post.Id = p.GeneratedKeys[0]
		
		tc.Posts = append(tc.Posts, post)
		insertedTerm, err := r.DB(db).Table(tcTable).Insert(tc).RunWrite(session)
		if err != nil {
			fmt.Println(err)
		}
		capsule = insertedTerm.GeneratedKeys[0]
		
	} else {
		creator := CheckTravelCapsuleExists(travelcapsule, session)
		fmt.Println("Creator: " + creator + " User: " + username)
		if creator == username {
			r.DB(db).Table(tcTable).Get(travelcapsule).
			Update(map[string]interface{}{"posts": r.Row.Field("posts").
			Append(post)}).
			RunWrite(session)
		}
	}
	
	
	return capsule
}
