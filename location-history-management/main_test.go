package main

import (
	"bytes"
	context "context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/mmilosevicgd/location-tracking/db"
	lhmp "github.com/mmilosevicgd/location-tracking/location-history-management/proto"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	locationHistoryManagementClient lhmp.GRPCClient

	bgCoordinates = []float64{20.2576593, 44.8154844}
	cuCoordinates = []float64{21.3135146, 43.9322129}
	deCoordinates = []float64{21.4197348, 44.0947626}
	jaCoordinates = []float64{21.2185255, 43.9754164}
	kgCoordinates = []float64{20.8244312, 44.0175876}
	pnCoordinates = []float64{21.344118, 43.8685548}
)

// initLocationHistoryManagementClient initializes the location history management client
func initLocationHistoryManagementClient() {
	if locationHistoryManagementClient != nil {
		log.Println("location history management client already initialized")
		return
	}

	client := lhmp.MustCreateClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	locationHistoryManagementClient = client
	log.Println("successfully initialized location history management client")
}

// disconnectLocationHistoryManagementClient disconnects the location history management client
func disconnectLocationHistoryManagementClient() {
	if locationHistoryManagementClient == nil {
		log.Println("location history management client is nil, skipping disconnection")
		return
	}

	log.Println("disconnecting location history management client...")

	if err := locationHistoryManagementClient.Close(); err != nil {
		log.Printf("error disconnecting location history management client: %v\n", err)

	} else {
		log.Println("successfully disconnected location history management client")
	}
}

func TestUserDistance(t *testing.T) {
	mongoClient = db.CreateMockDBClient()
	go main()
	time.Sleep(2 * time.Second)
	initLocationHistoryManagementClient()
	defer disconnectLocationHistoryManagementClient()

	type locationInfo struct {
		coordinates []float64
		timestamp   string
	}

	testData := []struct {
		username     string
		locationInfo []locationInfo
		start        string
		end          string
		firstAfter   int
		lastBefore   int
		expected     float64
	}{
		{
			username:     "user1",
			locationInfo: []locationInfo{},
			start:        "2025-01-01T00:00:00+00:00",
			end:          "2025-01-02T00:00:00+00:00",
			firstAfter:   -1,
			lastBefore:   -1,
			expected:     0.0,
		},
		{
			username: "user2",
			locationInfo: []locationInfo{
				{coordinates: deCoordinates, timestamp: "2025-01-03T00:00:00+00:00"},
				{coordinates: bgCoordinates, timestamp: "2025-01-04T00:00:00+00:00"},
				{coordinates: cuCoordinates, timestamp: "2025-01-05T00:00:00+00:00"},
				{coordinates: jaCoordinates, timestamp: "2025-01-06T00:00:00+00:00"},
			},
			start:      "2025-01-07T00:00:00+00:00",
			end:        "2025-01-08T00:00:00+00:00",
			firstAfter: -1,
			lastBefore: -1,
			expected:   0.0,
		},
		{
			username: "user3",
			locationInfo: []locationInfo{
				{coordinates: bgCoordinates, timestamp: "2025-01-10T00:00:00+00:00"},
				{coordinates: cuCoordinates, timestamp: "2025-01-11T00:00:00+00:00"},
				{coordinates: jaCoordinates, timestamp: "2025-01-12T00:00:00+00:00"},
				{coordinates: kgCoordinates, timestamp: "2025-01-13T00:00:00+00:00"},
			},
			start:      "2025-01-09T00:00:00+00:00",
			end:        "2025-01-11T00:00:00+00:00",
			firstAfter: 0,
			lastBefore: 1,
			expected:   129.183169,
		},
		{
			username: "user4",
			locationInfo: []locationInfo{
				{coordinates: bgCoordinates, timestamp: "2025-01-14T00:00:00+00:00"},
				{coordinates: cuCoordinates, timestamp: "2025-01-15T00:00:00+00:00"},
				{coordinates: jaCoordinates, timestamp: "2025-01-16T00:00:00+00:00"},
				{coordinates: kgCoordinates, timestamp: "2025-01-17T00:00:00+00:00"},
				{coordinates: pnCoordinates, timestamp: "2025-01-18T00:00:00+00:00"},
			},
			start:      "2025-01-14T01:00:00+00:00",
			end:        "2025-01-17T01:00:00+00:00",
			firstAfter: 1,
			lastBefore: 3,
			expected:   40.865308,
		},
	}

	for _, singleTestData := range testData {
		for i := range singleTestData.locationInfo {
			if i > 0 {
				err := setCurrentLocationInfo(singleTestData.username, singleTestData.locationInfo[i-1].timestamp)

				if err != nil {
					t.Fatalf("error setting current location info: %v\n", err)
					break
				}
			}

			err := updateUserLocation(singleTestData.username, singleTestData.locationInfo[i].coordinates, singleTestData.locationInfo[i].timestamp)

			if err != nil {
				t.Fatalf("error updating user location: %v", err)
				break
			}
		}

		if singleTestData.firstAfter != -1 {
			err := setAfterLocationInfo(singleTestData.username, singleTestData.locationInfo[singleTestData.firstAfter].timestamp, singleTestData.start)

			if err != nil {
				t.Fatalf("error setting after location info: %v", err)
				break
			}
		}

		if singleTestData.lastBefore != -1 {
			err := setBeforeLocationInfo(singleTestData.username, singleTestData.locationInfo[singleTestData.lastBefore].timestamp, singleTestData.end)

			if err != nil {
				t.Fatalf("error setting before location info: %v", err)
				continue
			}
		}

		distance, err := getDistance(singleTestData.username, singleTestData.start, singleTestData.end)

		if err != nil {
			t.Fatalf("error getting distance: %v", err)
			continue
		}

		if math.Abs(distance-singleTestData.expected) > 0.001 {
			t.Errorf("expected distance %f, got %f", singleTestData.expected, distance)
		}
	}
}

func setCurrentLocationInfo(username, timestamp string) error {
	parsedTimestamp, err := time.Parse(time.RFC3339, timestamp)

	if err != nil {
		return fmt.Errorf("error parsing timestamp '%s': %v\n", timestamp, err)
	}

	cursor := mongoClient.(db.MockDBClient).GetResponse(locationHistoryCollection, bson.M{
		"username":  username,
		"timestamp": parsedTimestamp.UnixMilli(),
	}, nil, nil, 0, 0)

	if cursor == nil {
		return fmt.Errorf("error getting cursor for username %s and timestamp '%s'\n", username, timestamp)
	}

	defer cursor.Close(context.Background())
	locations := []any{}
	err = cursor.All(context.Background(), &locations)

	if err != nil {
		return fmt.Errorf("error getting all documents: %v\n", err)
	}

	if len(locations) != 1 {
		return fmt.Errorf("expected 1 document, got %d\n", len(locations))
	}

	mongoClient.(db.MockDBClient).SetResponse(locationHistoryCollection, bson.M{
		"username": username,
	}, bson.M{
		"distance": 1,
		"location": 1,
	}, bson.M{
		"timestamp": -1,
	}, 1, 1, locations)

	return nil
}

func setAfterLocationInfo(username, timestamp, after string) error {
	parsedTimestamp, err := time.Parse(time.RFC3339, timestamp)

	if err != nil {
		return fmt.Errorf("error parsing timestamp '%s': %v", timestamp, err)
	}

	cursor := mongoClient.(db.MockDBClient).GetResponse(locationHistoryCollection, bson.M{
		"username":  username,
		"timestamp": parsedTimestamp.UnixMilli(),
	}, nil, nil, 0, 0)

	if cursor == nil {
		return fmt.Errorf("error getting cursor for username %s and timestamp '%s'\n", username, timestamp)
	}

	defer cursor.Close(context.Background())
	locations := []any{}
	err = cursor.All(context.Background(), &locations)

	if err != nil {
		return fmt.Errorf("error getting all documents: %v\n", err)
	}

	if len(locations) != 1 {
		return fmt.Errorf("expected 1 document, got %d\n", len(locations))
	}

	parsedAfterTimestamp, err := time.Parse(time.RFC3339, after)

	if err != nil {
		return fmt.Errorf("error parsing timestamp '%s': %v\n", timestamp, err)
	}

	mongoClient.(db.MockDBClient).SetResponse(locationHistoryCollection, bson.M{
		"username": username,
		"timestamp": bson.M{
			"$gte": parsedAfterTimestamp.UnixMilli(),
		},
	}, bson.M{
		"distance": 1,
	}, bson.M{
		"timestamp": 1,
	}, 1, 1, locations)

	return nil
}

func setBeforeLocationInfo(username, timestamp, before string) error {
	parsedTimestamp, err := time.Parse(time.RFC3339, timestamp)

	if err != nil {
		return fmt.Errorf("error parsing timestamp '%s': %v\n", timestamp, err)
	}

	cursor := mongoClient.(db.MockDBClient).GetResponse(locationHistoryCollection, bson.M{
		"username":  username,
		"timestamp": parsedTimestamp.UnixMilli(),
	}, nil, nil, 0, 0)

	if cursor == nil {
		return fmt.Errorf("error getting cursor for username %s and timestamp '%s'", username, timestamp)
	}

	defer cursor.Close(context.Background())
	locations := []any{}
	err = cursor.All(context.Background(), &locations)

	if err != nil {
		return fmt.Errorf("error getting all documents: %v\n", err)
	}

	if len(locations) != 1 {
		return fmt.Errorf("expected 1 document, got %d", len(locations))
	}

	parsedBeforeTimestamp, err := time.Parse(time.RFC3339, before)

	if err != nil {
		return fmt.Errorf("error parsing timestamp '%s': %v\n", timestamp, err)
	}

	mongoClient.(db.MockDBClient).SetResponse(locationHistoryCollection, bson.M{
		"username": username,
		"timestamp": bson.M{
			"$lte": parsedBeforeTimestamp.UnixMilli(),
		},
	}, bson.M{
		"distance": 1,
	}, bson.M{
		"timestamp": -1,
	}, 1, 1, locations)

	return nil
}

func updateUserLocation(username string, coordinates []float64, timestamp string) error {
	parsedTimestamp, err := time.Parse(time.RFC3339, timestamp)

	if err != nil {
		return fmt.Errorf("error parsing timestamp '%s': %v\n", timestamp, err)
	}

	_, err = locationHistoryManagementClient.UpdateUserLocation(context.Background(), &lhmp.LocationInfo{
		Username: username,
		Location: &lhmp.Location{
			Type:        "Point",
			Coordinates: coordinates,
		},
		Timestamp: parsedTimestamp.UnixMilli(),
	})

	if err != nil {
		return fmt.Errorf("error updating user location: %v\n", err)
	}

	return nil
}

func getDistance(username, start, end string) (float64, error) {
	payload, err := json.Marshal(struct {
		Username string `json:"username"`
		Start    string `json:"start"`
		End      string `json:"end"`
	}{
		Username: username,
		Start:    start,
		End:      end,
	})

	if err != nil {
		return 0, fmt.Errorf("error marshaling payload: %v\n", err)
	}

	resp, err := http.Post("http://localhost:8080/user/distance", "application/json", bytes.NewBuffer(payload))

	if err != nil {
		return 0, fmt.Errorf("error making post request: %v", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return 0, fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("expected status code 200, got %d", resp.StatusCode)
	}

	distance := struct {
		Distance float64 `json:"distance"`
	}{}

	err = json.Unmarshal(body, &distance)

	if err != nil {
		return 0, fmt.Errorf("error unmarshaling response body: %v", err)
	}

	return distance.Distance, nil
}
