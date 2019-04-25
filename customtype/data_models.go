package customtype

import (
	"time"
	"gopkg.in/rethinkdb/rethinkdb-go.v5/types"
)

// Different relation types
const (
	LikeType     = 1
	FollowerType = 2
)

// Destination and source types for relations
const (
	ProfilePicType = 4
	PostType       = 5
	TCType         = 6
	UserType       = 7
	CommentType    = 8
)

// User struct represent the user
type User struct {
	FName      string `rethinkdb:"fname"`
	LName      string `rethinkdb:"lname"`
	UName      string `rethinkdb:"username"`
	Salt       string `rethinkdb:"salt"`
	Passwd     string `rethinkdb:"password"`
	Email      string `rethinkdb:"email"`
	Verified   bool   `rethinkdb:"verified"`
	ProfilePic Image  `rethinkdb:"profile_pic"`
}

// Relation represents the one-many relation between user and post/travelcapsule
// Many-one has to be imposed
// Type can be LikeType or FollowerType
type Relation struct {
	Id string `rethinkdb:"id,omitempty"`
	Src       string    `rethinkdb:"src"`
	Dest      string    `rethinkdb:"dest"`
	CreatedOn time.Time `rethinkdb:"created_on"`
	Weight    float32   `rethinkdb:"weight"`
	Type      int       `rethinkdb:"type"`
}

// Body represents the message body in posts and comments
type Body struct {
	Message string `rethinkdb:"message"`
	Img     Image  `rethinkdb:"image"`
}

// Post can be made up of text, images or links.
// This is a parent type of TravelCapsule
type Post struct {
	Id string `rethinkdb:"id,omitempty"`
	Title     string    `rethinkdb:"title"`
	CreatedOn time.Time `rethinkdb:"created_on"`
	CreatedBy string    `rethinkdb:"created_by"`
	PostBody  Body      `rethinkdb:"body"`
	Hashtags  []string  `rethinkdb:"hashtags"`
	Likes     int       `rethinkdb:"likes"`
	Location  types.Point `rethinkdb:"location"`
	Place	  string	`rethinkdb:"place"`
}

// SimplePost represents the barebones data required to render it
// on the app
type SimplePost struct {
	Id 		  string 	`rethinkdb:"id,omitempty"`
	Title     string	`rethinkdb:"title"`
	Message   string 	
	Image	  string
	CreatedOn time.Time `rethinkdb:"created_on"`
	Place	  string	`rethinkdb:"place"`
}

// Comment struct represents the relationship between the user, comment
// and the post
type Comment struct {
	CommentBody  Body   `rethinkdb:"body"`
	CreatedOn time.Time `rethinkdb:"created_on"`
	CreatedBy string    `rethinkdb:"created_by"`
	Parent    string    `rethinkdb:"parent"`
	Likes     int       `rethinkdb:"likes"`
}

// TravelCapsule struct represents a collection of posts that can also be
// a standalone post
type TravelCapsule struct {
	Id		  string	`rethinkdb:"id,omitempty"`
	Title 	  string    `rethinkdb:"title"`
	Posts     []string  `rethinkdb:"posts"`
	CreatedOn time.Time `rethinkdb:"created_on"`
	CreatedBy string    `rethinkdb:"created_by"`
	UpdatedOn time.Time `rethinkdb:"updated_on"`
	Hashtags  []string  `rethinkdb:"hashtags"`
	Likes     int       `rethinkdb:"likes"`
}

// Hashtag struct represents the relation between hastags and posts
type Hashtag struct {
	Hashtag string `rethinkdb:"hashtag"`
	Parent  Post   `rethinkdb:"parent"`
}

// Image represents the images found in posts, comments and profile pics
type Image struct {
	Link         string    `rethinkdb:"url"`
	CreatedOn    string    `rethinkdb:"created_on"`
	UploadedOn   time.Time `rethinkdb:"uploaded_on"`
	Manufacturer string    `rethinkdb:"manufacturer"`
	Model        string    `rethinkdb:"model"`
	// Location     types.Geometry "rethink:`location`"
}
