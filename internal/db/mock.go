package db

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MockDBClient struct {
	responses map[string][]any
}

func (m MockDBClient) Disconnect() error {
	return nil
}

func (m MockDBClient) CreateCollection(collectionName string) error {
	return nil
}

func (m MockDBClient) MustCreateCollection(collectionName string) {
	// No-op for mock
}

func (m MockDBClient) SaveOrReplaceDocument(collectionName string, document any, filter map[string]any) error {
	m.SetResponse(collectionName, filter, nil, nil, 0, 0, []any{document})
	return nil
}

func (m MockDBClient) CreateIndex(collectionName, field string, sort int) error {
	return nil
}

func (m MockDBClient) MustCreateIndex(collectionName, field string, sort int) {
	// No-op for mock
}

func (m MockDBClient) Create2dSphereIndex(collectionName, field string) error {
	return nil
}

func (m MockDBClient) MustCreate2dSphereIndex(collectionName, field string) {
	// No-op for mock
}

func (m MockDBClient) Find(collectionName string, filter, projection, sort map[string]any, pageNumber, pageSize int) (*mongo.Cursor, error) {
	return m.GetResponse(collectionName, filter, projection, sort, pageNumber, pageSize), nil
}

// SetResponse sets the response for the given collection name, filter, projection, sort, page number, and page size
func (m MockDBClient) SetResponse(collectionName string, filter, projection, sort map[string]any, pageNumber, pageSize int, result []any) {
	m.responses[generateKey(collectionName, filter, projection, sort, pageNumber, pageSize)] = result
}

// GetResponse retrieves the response for the given collection name, filter, projection, sort, page number, and page size
func (m MockDBClient) GetResponse(collectionName string, filter, projection, sort map[string]any, pageNumber, pageSize int) *mongo.Cursor {
	key := generateKey(collectionName, filter, projection, sort, pageNumber, pageSize)
	cursor, err := mongo.NewCursorFromDocuments(m.responses[key], nil, nil)

	if err != nil {
		log.Fatalf("failed to create cursor for collection '%s', filter '%v', projection '%v' and sort '%v': %v", collectionName, filter, projection, sort, err)
	}

	return cursor
}

// generateKey generates a unique key for the given collection name, filter, projection, sort, page number, and page size
func generateKey(collectionName string, filter, projection, sort map[string]any, pageNumber, pageSize int) string {
	jsonAsString, err := json.Marshal(map[string]any{
		"collectionName": collectionName,
		"filter":         filter,
		"projection":     projection,
		"sort":           sort,
		"pageNumber":     pageNumber,
		"pageSize":       pageSize,
	})

	if err != nil {
		log.Fatalf("failed to generate key for collection '%s', filter '%v', projection '%v' and sort '%v': %v", collectionName, filter, projection, sort, err)
	}

	key, err := hashJSON(jsonAsString)

	if err != nil {
		log.Fatalf("failed to hash JSON for collection '%s', filter '%v', projection '%v' and sort '%v': %v", collectionName, filter, projection, sort, err)
	}

	return string(key)
}

// hashJSON calculates a hash of a json object, independent of the order of fields.
func hashJSON(input []byte) (string, error) {
	var obj map[string]interface{}

	if err := json.Unmarshal(input, &obj); err != nil {
		return "", fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	canonicalJSON, err := json.Marshal(obj)

	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %v", err)
	}

	hash := sha256.Sum256(canonicalJSON)
	return fmt.Sprintf("%x", hash), nil
}

// CreateMockDBClient creates a new mock db client
func CreateMockDBClient() MockDBClient {
	return MockDBClient{
		responses: map[string][]any{},
	}
}
