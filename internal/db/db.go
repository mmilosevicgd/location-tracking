package db

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ClientInfo struct {
	AuthSource      string
	Username        string
	Password        string
	Uri             string
	DefaultDatabase string
}

type DBClient interface {
	Disconnect() error
	CreateCollection(collectionName string) error
	MustCreateCollection(collectionName string)
	SaveOrReplaceDocument(collectionName string, document any, filter map[string]any) error
	CreateIndex(collectionName, field string, sort int) error
	MustCreateIndex(collectionName, field string, sort int)
	Create2dSphereIndex(collectionName, field string) error
	MustCreate2dSphereIndex(collectionName, field string)
	Find(collectionName string, filter, projection, sort map[string]any, pageNumber, pageSize int) (*mongo.Cursor, error)
}

type MongoClient struct {
	client    *mongo.Client
	defaultDb *mongo.Database
}

// Disconnect closes the mongodb connection
func (mc *MongoClient) Disconnect() error {
	return mc.client.Disconnect(context.Background())
}

// CreateCollection creates a new collection in the mongodb database
func (mc *MongoClient) CreateCollection(collectionName string) error {
	return mc.defaultDb.CreateCollection(context.Background(), collectionName)
}

// MustCreateCollection creates a new collection in the mongodb database and panics if it fails
func (mc *MongoClient) MustCreateCollection(collectionName string) {
	if err := mc.CreateCollection(collectionName); err != nil {
		log.Fatalf("failed to create collection '%s': %v\n", collectionName, err)
	}
}

// SaveOrReplaceDocument saves or replaces a document in the mongodb collection
func (mc *MongoClient) SaveOrReplaceDocument(collectionName string, document any, filter map[string]any) error {
	options := options.Replace().SetUpsert(true)
	_, err := mc.defaultDb.Collection(collectionName).ReplaceOne(context.Background(), filter, document, options)

	return err
}

// CreateIndex creates an index on the specified field in the mongodb collection with the specified sort order
func (mc *MongoClient) CreateIndex(collectionName, field string, sort int) error {
	indexModel := mongo.IndexModel{
		Keys: bson.M{
			field: sort,
		},
	}

	collection := mc.defaultDb.Collection(collectionName)
	_, err := collection.Indexes().CreateOne(context.Background(), indexModel)

	return err
}

// MustCreateIndex creates an index on the specified field in the mongodb collection with the specified sort order and panics if it fails
func (mc *MongoClient) MustCreateIndex(collectionName, field string, sort int) {
	if err := mc.CreateIndex(collectionName, field, sort); err != nil {
		log.Fatalf("failed to create index on field '%s' in collection '%s': %v\n", field, collectionName, err)
	}
}

// Create2dSphereIndex creates a 2dsphere index on the specified field in the mongodb collection
func (mc *MongoClient) Create2dSphereIndex(collectionName, field string) error {
	indexModel := mongo.IndexModel{
		Keys: bson.M{
			field: "2dsphere",
		},
	}

	collection := mc.defaultDb.Collection(collectionName)
	_, err := collection.Indexes().CreateOne(context.Background(), indexModel)

	return err
}

// MustCreate2dSphereIndex creates a 2dsphere index on the specified field in the mongodb collection and panics if it fails
func (mc *MongoClient) MustCreate2dSphereIndex(collectionName, field string) {
	if err := mc.Create2dSphereIndex(collectionName, field); err != nil {
		log.Fatalf("failed to create 2dsphere index on field '%s' in collection '%s': %v\n", field, collectionName, err)
	}
}

// Find retrieves documents from the mongodb collection based on the specified filter, projection, and sort options and paginates the results
func (mc *MongoClient) Find(collectionName string, filter, projection, sort map[string]any, pageNumber, pageSize int) (*mongo.Cursor, error) {
	collection := mc.defaultDb.Collection(collectionName)
	options := options.Find()

	if sort != nil {
		options.SetSort(sort)
	}

	if projection != nil {
		options.SetProjection(projection)
	}

	options.SetLimit(int64(pageSize)).SetSkip(int64(pageSize * (pageNumber - 1)))
	return collection.Find(context.Background(), filter, options)
}

// CreateClient creates a new mongodb client with the specified client info
func CreateClient(clientInfo ClientInfo) (*MongoClient, error) {
	auth := options.Credential{
		AuthSource: clientInfo.AuthSource,
		Username:   clientInfo.Username,
		Password:   clientInfo.Password,
	}

	opts := options.Client().ApplyURI(clientInfo.Uri).SetAuth(auth)
	client, err := mongo.Connect(opts)

	if err != nil {
		return nil, err
	}

	return &MongoClient{
		client:    client,
		defaultDb: client.Database(clientInfo.DefaultDatabase),
	}, nil
}

// MustCreateClient creates a new mongodb client with the specified client info and panics if it fails
func MustCreateClient(clientInfo ClientInfo) *MongoClient {
	mongoClient, err := CreateClient(clientInfo)

	if err != nil {
		log.Fatalf("failed to create mongodb client for uri '%s': %v\n", clientInfo.Uri, err)
	}

	return mongoClient
}
