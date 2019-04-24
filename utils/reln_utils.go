package utils

import (
	ct "github.com/cvhariharan/Utils/customtype"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
	"time"
	"os"
)


// AddComment takes in a postid and username and inserts a comment into that
// post from the user. Returns true if successful, else false
func AddComment(postId, username, message string, session *r.Session) bool {
	db := os.Getenv("DB")
	commentTable := os.Getenv("COMMENTTABLE")

	if CheckPostExists(postId, session) {
		commentBody := ct.Body{
			Message: message,
		}
		comment := ct.Comment{
			CommentBody: commentBody,
			CreatedOn: time.Now(),
			CreatedBy: username,
			Parent: postId,
			Likes: 0,
		}

		r.DB(db).Table(commentTable).Insert(comment).Run(session)
		return true
	}

	return false
	
}

// GetComments takes in a postid and returns all its comments as
// a comments Array
func GetComments(postId string, session *r.Session) []ct.Comment {
	var comments []ct.Comment
	var comment ct.Comment
	db := os.Getenv("DB")
	commentTable := os.Getenv("COMMENTTABLE")

	cur, _ := r.DB(db).Table(commentTable).GetAllByIndex("parent", postId).Run(session)

	for cur.Next(&comment) {
		comments = append(comments, comment)
	}
	return comments
}


// CheckRelationExists takes a src and dest and checks if they are connected
// by the relation table
func CheckRelationExists(src string, dest string, rType int, session *r.Session) bool {
	var u ct.Relation
	db := os.Getenv("DB")
	table := os.Getenv("RELNTABLE")
	cur, _ := r.DB(db).Table(table).GetAllByIndex("src", src).Run(session)

	for cur.Next(&u) {
		if u.Dest == dest && u.Type == rType {
			return true
		}
	}
	// _ = cur.One(&u)
	cur.Close()
	return false
}

// GetFollowees returns an array of userids of people followed
// by the user
func GetFollowees(username string, session *r.Session) []string {
	var followees []string
	var u ct.Relation
	db := os.Getenv("DB")
	table := os.Getenv("RELNTABLE")
	cur, _ := r.DB(db).Table(table).GetAllByIndex("src", username).Run(session)

	for cur.Next(&u) {
		// fmt.Println(u)
		if u.Type == ct.FollowerType {
			followees = append(followees, u.Dest)
		}
	}

	return followees
}

// GetFollowers returns the followers of the user as a list of usernames
func GetFollowers(username string, session *r.Session) []string {
	var followers []string
	var u ct.Relation
	db := os.Getenv("DB")
	table := os.Getenv("RELNTABLE")
	cur, _ := r.DB(db).Table(table).GetAllByIndex("dest", username).Run(session)
	
	for cur.Next(&u) {
		// fmt.Println(u)
		if u.Type == ct.FollowerType {
			followers = append(followers, u.Dest)
		}
	}

	return followers
}

func getRelation(src string, dest string, rType int, session *r.Session) ct.Relation {
	var u ct.Relation
	var t ct.Relation
	db := os.Getenv("DB")
	table := os.Getenv("RELNTABLE")
	cur, _ := r.DB(db).Table(table).GetAllByIndex("src", src).Filter(r.Row.Field("dest").Eq(dest)).Run(session)

	for cur.Next(&u) {
		if u.Type == rType {
			return u
		}
	}
	return t
}

// FollowUser takes follower and followee usernames, checks if they are already related and
// if not, creates a reln between them
func FollowUser(follower string, followee string, session *r.Session) bool {
	userTable := os.Getenv("USERTABLE")
	relationTable := os.Getenv("RELNTABLE")
	db := os.Getenv("DB")
	if CheckUserExists(follower, userTable, session) && CheckUserExists(followee, userTable, session) && follower != followee {
		// Check if user already follows the followee
		if !CheckRelationExists(follower, followee, ct.FollowerType, session) {
			rel := ct.Relation{
				Src:       follower,
				Dest:      followee,
				CreatedOn: time.Now(),
				Type:      ct.FollowerType,
			}
			r.DB(db).Table(relationTable).Insert(rel).Run(session)
			return true
		}
	}
	return false
}


// Unfollowuser takes in a user and a followee of the user and unfollows the followee.
// Returns true if successful else false
func UnfollowUser(follower string, followee string, session *r.Session) bool {
	userTable := os.Getenv("USERTABLE")
	relationTable := os.Getenv("RELNTABLE")
	db := os.Getenv("DB")
	if CheckUserExists(follower, userTable, session) && CheckUserExists(followee, userTable, session) && follower != followee {
		if CheckRelationExists(follower, followee, ct.FollowerType, session) {
			r.DB(db).Table(relationTable).GetAllByIndex("src", follower).
			Filter(r.Row.Field("dest").Eq(followee)).
			Filter(r.Row.Field("type").Eq(ct.FollowerType)).Delete().Run(session)
			return true
		}
	}
	return false
}

// LikePost takes in a post id and username and creates
// a like relation between them andreturns true if successful
func LikePost(postId string, username string, session *r.Session) bool {
	db := os.Getenv("DB")
	relationTable := os.Getenv("RELNTABLE")
	postTable := os.Getenv("POSTTABLE")
	if !CheckRelationExists(username, postId, ct.LikeType, session) && CheckPostExists(postId, session) {
		rel := ct.Relation{
			Src: username,
			Dest: postId,
			CreatedOn: time.Now(),
			Type: ct.LikeType,
		}
		r.DB(db).Table(relationTable).Insert(rel).Run(session)
		// Update the like count on the post as well
		r.DB(db).Table(postTable).Get(postId).Update(map[string]interface{}{
			"likes": r.Row.Field("likes").Add(1),
		}).Run(session)
		return true
	}
	return false
}

// UnlikePost takes in post id and username and unlikes the post
// if the user had liked it
func UnlikePost(postId string, username string, session *r.Session) bool {
	var t ct.Relation
	db := os.Getenv("DB")
	relationTable := os.Getenv("RELNTABLE")
	postTable := os.Getenv("POSTTABLE")
	if CheckRelationExists(username, postId, ct.LikeType, session) {
		u := getRelation(username, postId, ct.LikeType, session)
		// Only if u is not an empty struct
		if u != t {
			r.DB(db).Table(relationTable).Get(u.Id).Delete().Run(session)

			// Update the likes count
			r.DB(db).Table(postTable).Get(postId).Update(map[string]interface{}{
				"likes": r.Row.Field("likes").Sub(1),
			}).Run(session)
			return true
		}
	}
	return false
}