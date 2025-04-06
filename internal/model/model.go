package model

type Location struct {
	Type        string    `bson:"type" json:"type"`
	Coordinates []float64 `bson:"coordinates" json:"coordinates"`
}

type LocationInfo struct {
	Username  string   `bson:"username" json:"username"`
	Location  Location `bson:"location" json:"location"`
	Distance  float64  `bson:"distance" json:"distance"`
	Timestamp int64    `bson:"timestamp" json:"timestamp"`
}