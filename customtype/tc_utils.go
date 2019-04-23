package customtype

// Functions that help in sorting travelcapsules by date
type TravelCapsules []TravelCapsule

func (tc TravelCapsules) Len() int {
    return len(tc)
}
func (tc TravelCapsules) Swap(i, j int) {
    tc[i], tc[j] = tc[j], tc[i]
}
func (tc TravelCapsules) Less(i, j int) bool {
    return tc[i].UpdatedOn.After(tc[j].UpdatedOn)
}