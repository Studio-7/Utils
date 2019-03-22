package customtype

import (
	"time"
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
	ProfilePic Image  `rethinkdb:"profile_pic"`
}

// Relation represents the one-many relation between user and post/travelcapsule
// Many-one has to be imposed
// Type can be LikeType or FollowerType
type Relation struct {
	Src       string    `rethinkdb:"src"`
	Dest      string    `rethinkdb:"dest"`
	CreatedOn time.Time `rethinkdb:"created_on"`
	Weight    float32   `rethinkdb:"weight"`
	Type      int       `rethinkdb:"type"`
}

// Body represents the message body in posts and comments
type Body struct {
	Message string `rethinkdb:"message"`
	Image   string `rethinkdb:"image"`
}

// Post can be made up of text, images or links.
// This is a parent type of TravelCapsule
type Post struct {
	Title     string    `rethinkdb:"title"`
	CreatedOn time.Time `rethinkdb:"created_on"`
	CreatedBy string    `rethinkdb:"created_by"`
	Body
	Hashtags []string `rethinkdb:"hashtags"`
}

// Comment struct represents the relationship between the user, comment
// and the post
type Comment struct {
	Body
	CreatedOn time.Time `rethinkdb:"created_on"`
	CreatedBy string    `rethinkdb:"created_by"`
	Parent    string    `rethinkdb:"parent"`
}

// TravelCapsule struct represents a collection of posts that can also be
// a standalone post
type TravelCapsule struct {
	Posts     []Post    `rethinkdb:"posts"`
	CreatedOn time.Time `rethinkdb:"created_on"`
	CreatedBy string    `rethinkdb:"created_by"`
	Hashtags  []string  `rethinkdb:"hashtags"`
}

// Hashtag struct represents the relation between hastags and posts
type Hashtag struct {
	Hashtag string `rethinkdb:"hashtag"`
	Parent  Post   `rethinkdb:"parent"`
}

// Image represents the images found in posts, comments and profile pics
type Image struct {
	Link         string    `rethinkdb:"url"`
	CreatedOn    time.Time `rethinkdb:"created_on"`
	UploadedOn   time.Time `rethinkdb:"uploaded_on"`
	Manufacturer string    `rethinkdb:"manufacturer"`
	Model        string    `rethinkdb:"model"`
	// Location     types.Geometry "rethink:`location`"
}
