package main

import (
	context "context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	pb "github.com/mmilosevicgd/location-tracking/location-history-management/proto"
	"github.com/mmilosevicgd/location-tracking/model"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/types/known/emptypb"
)

// calculateUserDistanceHandler validates the request data, extracts the username and timestamps, and calculates the distance traveled by the user between the two timestamps in kilometers
func calculateUserDistanceHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Username string `json:"username" validate:"required,alphanum,min=4,max=16"`
		Start    string `json:"start" validate:"required,customdatetime"`
		End      string `json:"end" validate:"required,customdatetime"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		log.Printf("error decoding request body: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := validate.Struct(data); err != nil {
		log.Printf("validation error for request data: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	distance, err := calculateUserDistance(data.Username, data.Start, data.End)

	if err != nil {
		log.Printf("error calculating user distance for username '%s' and date range '%s' - '%s': %v\n", data.Username, data.Start, data.End, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := struct {
		Distance float64 `json:"distance"`
	}{
		Distance: distance,
	}

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		log.Printf("error encoding response: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
}

// calculateUserDistance calculates the distance traveled by a user between two timestamps and returns the result
func calculateUserDistance(username, start, end string) (float64, error) {
	parsedStart, err := time.Parse(time.RFC3339, start)

	if err != nil {
		log.Printf("error parsing start time '%s' for username '%s': %v\n", start, username, err)
		return 0, err
	}

	initialDistance, err := getFirstAfter(username, parsedStart.UnixMilli())

	if err != nil {
		log.Printf("error retrieving first location after start time '%s' for username '%s': %v", start, username, err)
		return 0, err
	}

	parsedEnd, err := time.Parse(time.RFC3339, end)

	if err != nil {
		log.Printf("error parsing end time '%s' for username '%s': %v", end, username, err)
		return 0, err
	}

	if parsedEnd.Before(parsedStart) {
		log.Printf("end time '%s' is before start time '%s' for username '%s'\n", end, start, username)
		return 0, nil
	}

	finalDistance, err := getLastBefore(username, parsedEnd.UnixMilli())

	if err != nil {
		log.Printf("error retrieving last location before end time '%s' for username '%s': %v", end, username, err)
		return 0, err
	}

	return finalDistance - initialDistance, nil
}

// getFirstAfter retrieves the first location after a given date for a specific user
func getFirstAfter(username string, date int64) (float64, error) {
	return getDate(username, date, "after")
}

// getLastBefore retrieves the last location before a given date for a specific user
func getLastBefore(username string, date int64) (float64, error) {
	return getDate(username, date, "before")
}

// getDate retrieves the location total distance based on the specified date and type (after or before)
func getDate(username string, date int64, dateType string) (float64, error) {
	comparator := "$gte"

	if dateType == "before" {
		comparator = "$lte"
	}

	filter := bson.M{
		"username":  username,
		"timestamp": bson.M{comparator: date},
	}

	projection := bson.M{
		"distance": 1,
	}

	sorter := 1

	if dateType == "before" {
		sorter = -1
	}

	sort := bson.M{
		"timestamp": sorter,
	}

	cursor, err := mongoClient.Find(locationHistoryCollection, filter, projection, sort, 1, 1)

	if err != nil {
		log.Printf("error executing database query for username '%s', date '%s' and date type '%s': %v\n", username, strconv.FormatInt(date, 10), dateType, err)
		return 0, err
	}

	defer cursor.Close(context.Background())
	locations := []struct {
		Distance float64 `bson:"distance" json:"distance"`
	}{}

	if err := cursor.All(context.Background(), &locations); err != nil {
		log.Printf("error decoding cursor results for username '%s', date '%s' and date type '%s': %v\n", username, strconv.FormatInt(date, 10), dateType, err)
		return 0, err
	}

	if len(locations) == 0 {
		return 0, nil
	}

	return locations[0].Distance, nil
}

// UpdateUserLocation updates the location of a user in the database
func (s *protoServer) UpdateUserLocation(ctx context.Context, in *pb.LocationInfo) (*emptypb.Empty, error) {
	locationInfo := model.LocationInfo{
		Username: in.Username,
		Location: model.Location{
			Type:        in.Location.Type,
			Coordinates: in.Location.Coordinates,
		},
		Timestamp: in.Timestamp,
	}

	currentLocation, currentDistance, ok, err := findCurrent(locationInfo.Username)

	if err != nil {
		log.Printf("error finding current location for username '%s': %v\n", locationInfo.Username, err)
		return nil, err
	}

	if !ok {
		locationInfo.Distance = 0

	} else {
		locationInfo.Distance = calculateDistance(currentLocation, locationInfo.Location, currentDistance)
	}

	filter := bson.M{
		"username":  locationInfo.Username,
		"timestamp": locationInfo.Timestamp,
	}

	if err := mongoClient.SaveOrReplaceDocument(locationHistoryCollection, locationInfo, filter); err != nil {
		log.Printf("error saving or replacing document for username '%s': %v\n", locationInfo.Username, err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// findCurrent retrieves the current location and total distance for a specific user
func findCurrent(username string) (model.Location, float64, bool, error) {
	filter := bson.M{
		"username": username,
	}

	projection := bson.M{
		"location": 1,
		"distance": 1,
	}

	sort := bson.M{
		"timestamp": -1,
	}

	cursor, err := mongoClient.Find(locationHistoryCollection, filter, projection, sort, 1, 1)

	if err != nil {
		log.Printf("error executing database query for username '%s': %v\n", username, err)
		return model.Location{}, 0, false, err
	}

	defer cursor.Close(context.Background())
	locations := []struct {
		Location model.Location `bson:"location" json:"location"`
		Distance float64        `bson:"distance" json:"distance"`
	}{}

	if err := cursor.All(context.Background(), &locations); err != nil {
		log.Printf("error decoding cursor results for username '%s': %v\n", username, err)
		return model.Location{}, 0, false, err
	}

	if len(locations) == 0 {
		return model.Location{}, 0, false, nil
	}

	return locations[0].Location, locations[0].Distance, true, nil
}

// degreesToRadians converts degrees to radians
func degreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}

// calculateDistance calculates the distance between two geographical locations using the haversine formula and adds the current total distance to it
func calculateDistance(start, end model.Location, currentDistance float64) float64 {
	if start.Coordinates == nil || len(start.Coordinates) == 0 {
		return 0
	}

	startLongitude := degreesToRadians(start.Coordinates[0])
	startLatitude := degreesToRadians(start.Coordinates[1])
	endLongitude := degreesToRadians(end.Coordinates[0])
	endLatitude := degreesToRadians(end.Coordinates[1])

	diffLon := endLongitude - startLongitude
	diffLat := endLatitude - startLatitude

	a := math.Pow(math.Sin(diffLat/2), 2) + math.Cos(startLatitude)*math.Cos(endLatitude)*math.Pow(math.Sin(diffLon/2), 2)
	return 6371*2*math.Atan2(math.Sqrt(a), math.Sqrt(1-a)) + currentDistance
}
