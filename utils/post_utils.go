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

func getImage(imgloc string) ct.Image {
	var img ct.Image
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
	return img
}

// CreatePost take params, creates a post object and inserts it into the post table
// created to prevent direct handling of data-model structs in the handlers
func CreatePost(title string, message string, imgloc string, hashtags []string, username string, session *r.Session) bool {
	db := os.Getenv("DB")
	postTable := os.Getenv("POSTTABLE")
	var img ct.Image
	if imgloc != "" {
		img = getImage(imgloc)
		body := ct.Body{
			Message: message,
			Img:     img,
		}
		post := ct.Post{
			Title:     title,
			CreatedOn: time.Now(),
			CreatedBy: username,
			PostBody:  body,
			Hashtags:  hashtags,
		}
		// Insert post into table
		r.DB(db).Table(postTable).Insert(post).Run(session)
		return true
	}
	return false
}
