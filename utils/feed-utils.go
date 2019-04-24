package utils

import (
	ct "github.com/cvhariharan/Utils/customtype"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
	"sort"
	"os"
)

// GetFeed returns a feed of TCs in chronological order
func GetFeed(username string, session *r.Session) []ct.TravelCapsule {
	var tcs []ct.TravelCapsule
	var tc ct.TravelCapsule
	db := os.Getenv("DB")
	tcTable := os.Getenv("TCTABLE")
	followees := GetFollowees(username, session)
	for _, id := range followees {
		cur, _ := r.DB(db).Table(tcTable).GetAllByIndex("created_by", id).Run(session)
		for cur.Next(&tc) {
			tcs = append(tcs, tc)
		}
	}
	
	sort.Sort(ct.TravelCapsules(tcs))
	return tcs
}