package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	lhmp "github.com/mmilosevicgd/location-tracking/location-history-management/proto"
	"github.com/mmilosevicgd/location-tracking/model"
	"go.mongodb.org/mongo-driver/bson"
)

// updateUserLocationHandler validates the request data, extracts coordinates and updates the user's location
func updateUserLocationHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Username    string `json:"username" validate:"required,alphanum,min=4,max=16"`
		Coordinates string `json:"coordinates" validate:"required,customcoordinates"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		log.Printf("error decoding request body: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := validate.Struct(data); err != nil {
		log.Printf("validation error for input data: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	coordinates, err := extractCoordinates(data.Coordinates)

	if err != nil {
		log.Printf("error extracting coordinates '%s': %v\n", data.Coordinates, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := updateUserLocation(data.Username, coordinates); err != nil {
		log.Printf("error updating user location for username '%s' and coordinates '%v': %v\n", data.Username, coordinates, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// updateUserLocation updates the user's location in the database and sends the update to the location history management service
func updateUserLocation(username string, coordinates []float64) error {
	locationInfo := model.LocationInfo{
		Username: username,
		Location: model.Location{
			Type:        "Point",
			Coordinates: coordinates,
		},
		Timestamp: time.Now().UnixMilli(),
	}

	filter := map[string]any{
		"username": locationInfo.Username,
	}

	if err := mongoClient.SaveOrReplaceDocument(locationCollection, locationInfo, filter); err != nil {
		log.Printf("error saving or replacing document in mongodb for username '%s': %v", locationInfo.Username, err)
		return err
	}

	_, err := locationHistoryManagementClient.UpdateUserLocation(context.Background(), &lhmp.LocationInfo{
		Username: locationInfo.Username,
		Location: &lhmp.Location{
			Type:        locationInfo.Location.Type,
			Coordinates: locationInfo.Location.Coordinates,
		},
		Timestamp: locationInfo.Timestamp,
	})

	if err != nil {
		log.Printf("error sending location update to grpc service for username '%s': %v\n", locationInfo.Username, err)
		return err
	}

	return nil
}

// searchUserLocationHandler validates the request data, extracts coordinates and searches for users within a specified distance and returns their usernames
func searchUserLocationHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Coordinates string  `json:"coordinates" validate:"required,customcoordinates"`
		Distance    float64 `json:"distance" validate:"required,gte=0"`
		PageNumber  int     `json:"pageNumber" validate:"required,gt=0"`
		PageSize    int     `json:"pageSize" validate:"required,gt=0"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		log.Printf("error decoding request body: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := validate.Struct(data); err != nil {
		log.Printf("validation error for input data: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	coordinates, err := extractCoordinates(data.Coordinates)

	if err != nil {
		log.Printf("error extracting coordinates '%s': %v\n", data.Coordinates, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	usernames, err := searchUserLocation(coordinates, data.Distance, data.PageNumber, data.PageSize)

	if err != nil {
		log.Printf("error searching user locations for coordinates '%v' and distance '%f': %v\n", coordinates, data.Distance, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := struct {
		Usernames []string `json:"usernames"`
	}{
		Usernames: usernames,
	}

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		log.Printf("error encoding response: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
}

// searchUserLocation searches for users within a specified distance from the given coordinates and returns their usernames
func searchUserLocation(coordinates []float64, distance float64, pageNumber, pageSize int) ([]string, error) {
	target := model.Location{
		Type:        "Point",
		Coordinates: coordinates,
	}

	usernames, err := findNear(target, distance, pageNumber, pageSize)

	if err != nil {
		return nil, err
	}

	return usernames, nil
}

// extractCoordinates extracts coordinates from a string and returns them as a slice of float64
func extractCoordinates(coordinatesString string) ([]float64, error) {
	coordinates := []float64{}

	for _, coordinateString := range strings.Split(strings.ReplaceAll(coordinatesString, " ", ""), ",") {
		coordinate, err := strconv.ParseFloat(coordinateString, 64)

		if err != nil {
			return nil, err
		}

		coordinates = append(coordinates, coordinate)
	}

	slices.Reverse(coordinates)

	if coordinates[0] < -180 || coordinates[0] > 180 {
		return nil, fmt.Errorf("longitude out of range: %f", coordinates[0])
	}

	if coordinates[1] < -90 || coordinates[1] > 90 {
		return nil, fmt.Errorf("latitude out of range: %f", coordinates[1])
	}

	return coordinates, nil
}

// findNear finds users within a specified distance from the target location and returns their usernames
func findNear(target model.Location, distance float64, pageNumber, pageSize int) ([]string, error) {
	filter := bson.M{
		"location": bson.M{
			"$near": bson.M{
				"$geometry":    target,
				"$maxDistance": distance,
			},
		},
	}

	projection := bson.M{
		"username": 1,
	}

	sort := bson.M{
		"username": 1,
	}

	cursor, err := mongoClient.Find(locationCollection, filter, projection, sort, pageNumber, pageSize)

	if err != nil {
		log.Printf("error executing mongodb find query: %v", err)
		return nil, err
	}

	defer cursor.Close(context.Background())
	rawUsernames := []struct {
		Username string `bson:"username" json:"username"`
	}{}

	if err := cursor.All(context.Background(), &rawUsernames); err != nil {
		log.Printf("error decoding mongodb cursor results: %v", err)
		return nil, err
	}

	usernames := []string{}

	for _, rawUsername := range rawUsernames {
		usernames = append(usernames, rawUsername.Username)
	}

	return usernames, nil
}
