package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/mmilosevicgd/location-tracking/db"
	lhmp "github.com/mmilosevicgd/location-tracking/location-history-management/proto"
	"github.com/mmilosevicgd/location-tracking/model"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	bgCoordinates = "44.8154844,20.2576593"
	cuCoordinates = "43.9322129,21.3135146"
	deCoordinates = "44.0947626,21.4197348"
	jaCoordinates = "43.9754164,21.2185255"
	kgCoordinates = "44.0175876,20.8244312"
	pnCoordinates = "43.8685548,21.344118"
)

func TestSearchUserLocation(t *testing.T) {
	mongoClient = db.CreateMockDBClient()
	locationHistoryManagementClient = lhmp.CreateMockGRPCClient()
	go main()
	time.Sleep(2 * time.Second)

	type locationInfo struct {
		username    string
		coordinates string
	}

	testData := []struct {
		userLocations []locationInfo
		coordinates   string
		distance      float64
		pageNumber    int
		pageSize      int
		expected      []string
	}{
		{
			userLocations: []locationInfo{},
			coordinates:   deCoordinates,
			distance:      0.1,
			pageNumber:    1,
			pageSize:      10,
			expected:      []string{},
		},
		{
			userLocations: []locationInfo{
				{username: "user1", coordinates: deCoordinates},
				{username: "user2", coordinates: jaCoordinates},
				{username: "user3", coordinates: bgCoordinates},
			},
			coordinates: deCoordinates,
			distance:    0.1,
			pageNumber:  1,
			pageSize:    10,
			expected:    []string{"user1"},
		},
		{
			userLocations: []locationInfo{
				{username: "user4", coordinates: deCoordinates},
				{username: "user5", coordinates: jaCoordinates},
				{username: "user6", coordinates: bgCoordinates},
			},
			coordinates: cuCoordinates,
			distance:    50.0,
			pageNumber:  1,
			pageSize:    10,
			expected:    []string{"user4", "user5"},
		},
		{
			userLocations: []locationInfo{
				{username: "user7", coordinates: deCoordinates},
				{username: "user8", coordinates: jaCoordinates},
				{username: "user9", coordinates: bgCoordinates},
			},
			coordinates: cuCoordinates,
			distance:    1.0,
			pageNumber:  1,
			pageSize:    10,
			expected:    []string{},
		},
		{
			userLocations: []locationInfo{
				{username: "user10", coordinates: deCoordinates},
				{username: "user11", coordinates: jaCoordinates},
				{username: "user12", coordinates: pnCoordinates},
				{username: "user10", coordinates: bgCoordinates},
				{username: "user12", coordinates: kgCoordinates},
				{username: "user11", coordinates: cuCoordinates},
				{username: "user11", coordinates: deCoordinates},
				{username: "user10", coordinates: deCoordinates},
			},
			coordinates: deCoordinates,
			distance:    10.0,
			pageNumber:  1,
			pageSize:    10,
			expected:    []string{"user10", "user11"},
		},
	}

	for _, singleTestData := range testData {
		for _, location := range singleTestData.userLocations {
			err := updateLocation(location.username, location.coordinates)

			if err != nil {
				t.Fatalf("error updating location: %v", err)
				continue
			}

			err = validateLocation(location.username, location.coordinates)

			if err != nil {
				t.Fatalf("error validating location: %v", err)
				continue
			}
		}

		type usernameInfo struct {
			Username string `json:"username"`
		}

		expected := []any{}

		for _, username := range singleTestData.expected {
			expected = append(expected, usernameInfo{Username: username})
		}

		parsedCoordinates, err := extractCoordinates(singleTestData.coordinates)

		if err != nil {
			t.Fatalf("error extracting coordinates: %v", err)
			continue
		}

		mongoClient.(db.MockDBClient).SetResponse(locationCollection, bson.M{
			"location": bson.M{
				"$near": bson.M{
					"$geometry":    bson.M{"type": "Point", "coordinates": parsedCoordinates},
					"$maxDistance": singleTestData.distance,
				},
			},
		}, bson.M{
			"username": 1,
		}, bson.M{
			"username": 1,
		},
			singleTestData.pageNumber,
			singleTestData.pageSize,
			expected,
		)

		users, err := searchUsers(singleTestData.coordinates, singleTestData.distance, singleTestData.pageNumber, singleTestData.pageSize)

		if err != nil {
			t.Fatalf("error searching users: %v", err)
			continue
		}

		if len(users) != len(singleTestData.expected) {
			t.Errorf("expected users %v, got %v\n", singleTestData.expected, users)
			continue
		}

		for i := range users {
			if users[i] != singleTestData.expected[i] {
				t.Errorf("expected users %v, got %v\n", singleTestData.expected, users)
				break
			}
		}
	}
}

func updateLocation(username, coordinates string) error {
	payload, err := json.Marshal(struct {
		Username    string `json:"username"`
		Coordinates string `json:"coordinates"`
	}{
		Username:    username,
		Coordinates: coordinates,
	})

	if err != nil {
		return fmt.Errorf("error marshaling payload: %v", err)
	}

	resp, err := http.Post("http://localhost:8080/user/location", "application/json", bytes.NewBuffer(payload))

	if err != nil {
		return fmt.Errorf("error making post request: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status code 200, got %d", resp.StatusCode)
	}

	return nil
}

func searchUsers(coordinates string, distance float64, pageNumber, pageSize int) ([]string, error) {
	payload, err := json.Marshal(struct {
		Coordinates string  `json:"coordinates"`
		Distance    float64 `json:"distance"`
		PageNumber  int     `json:"pageNumber"`
		PageSize    int     `json:"pageSize"`
	}{
		Coordinates: coordinates,
		Distance:    distance,
		PageNumber:  pageNumber,
		PageSize:    pageSize,
	})

	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %v", err)
	}

	resp, err := http.Post("http://localhost:8080/user/search", "application/json", bytes.NewBuffer(payload))

	if err != nil {
		return nil, fmt.Errorf("error making post request: %v", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected status code 200, got %d", resp.StatusCode)
	}

	usernames := struct {
		Usernames []string `json:"usernames"`
	}{}

	err = json.Unmarshal(body, &usernames)

	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response body: %v", err)
	}

	return usernames.Usernames, nil
}

func validateLocation(username, coordinates string) error {
	cursor := mongoClient.(db.MockDBClient).GetResponse(locationCollection, bson.M{"username": username}, nil, nil, 0, 0)

	if cursor == nil {
		return fmt.Errorf("error getting cursor for username %s", username)
	}

	defer cursor.Close(context.Background())
	locations := []model.LocationInfo{}
	err := cursor.All(context.Background(), &locations)

	if err != nil {
		return fmt.Errorf("error getting all documents: %v", err)
	}

	if len(locations) != 1 {
		return fmt.Errorf("expected 1 document, got %d", len(locations))
	}

	extractedCoordinates, err := extractCoordinates(coordinates)

	if err != nil {
		return fmt.Errorf("error extracting coordinates: %v", err)
	}

	if locations[0].Username != username {
		return fmt.Errorf("expected username %s, got %s", username, locations[0].Username)
	}

	if len(locations[0].Location.Coordinates) != len(extractedCoordinates) {
		return fmt.Errorf("expected %d coordinates, got %d", len(extractedCoordinates), len(locations[0].Location.Coordinates))
	}

	for i := range locations[0].Location.Coordinates {
		if locations[0].Location.Coordinates[i] != extractedCoordinates[i] {
			return fmt.Errorf("expected coordinates %v, got %v", extractedCoordinates, locations[0].Location.Coordinates)
		}
	}

	if locations[0].Location.Type != "Point" {
		return fmt.Errorf("expected type point, got %s", locations[0].Location.Type)
	}

	if locations[0].Distance != 0 {
		return fmt.Errorf("expected distance 0, got %f", locations[0].Distance)
	}

	return nil
}
