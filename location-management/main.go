package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mmilosevicgd/location-tracking/db"
	lhmp "github.com/mmilosevicgd/location-tracking/location-history-management/proto"
	"github.com/mmilosevicgd/location-tracking/validation"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-playground/validator/v10"
)

const (
	locationCollection = "location"
)

var (
	validate                            = validator.New()
	mongoClient                         db.DBClient
	httpServer                          *http.Server
	locationHistoryManagementClient     lhmp.GRPCClient
)

func main() {
	go initValidations()
	go initMongoClient()
	go initLocationHistoryManagementClient()
	go initHttpServer()

	shutdown, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-shutdown.Done()

	wg := sync.WaitGroup{}
	wg.Add(3)
	go disconnectMongoClient(&wg)
	go disconnectLocationHistoryManagementClient(&wg)
	go shutdownHttpServer(&wg)
	wg.Wait()
}

// initValidations initializes the custom validations for the validator package
func initValidations() {
	validation.RegisterCustomValidations(validate)
}

// initMongoClient initializes the mongodb client and creates the necessary collections and indexes
func initMongoClient() {
	if mongoClient != nil {
		log.Println("mongo client already initialized")
		return
	}

	mongoClient = db.MustCreateClient(db.ClientInfo{
		AuthSource:      os.Getenv("MONGODB_AUTH_DB"),
		Username:        os.Getenv("MONGODB_USERNAME"),
		Password:        os.Getenv("MONGODB_PASSWORD"),
		Uri:             os.Getenv("MONGODB_URI"),
		DefaultDatabase: os.Getenv("MONGODB_DEFAULT_DB"),
	})

	mongoClient.MustCreateCollection(locationCollection)
	mongoClient.MustCreateIndex(locationCollection, "username", 1)
	mongoClient.MustCreate2dSphereIndex(locationCollection, "location")
	log.Println("successfully initialized mongo client and created collections and indexes")
}

// initLocationHistoryManagementClient initializes the location history management client
func initLocationHistoryManagementClient() {
	if locationHistoryManagementClient != nil {
		log.Println("location history management client already initialized")
		return
	}

	client := lhmp.MustCreateClient(os.Getenv("LOCATION_HISTORY_MANAGEMENT_GRPC_URI"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	locationHistoryManagementClient = client
	log.Println("successfully initialized location history management client")
}

// initHttpServer initializes the HTTP server and sets up the routes
func initHttpServer() {
	if httpServer != nil {
		log.Println("http server already initialized")
		return
	}

	log.Println("initializing http server...")
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("POST /user/location", updateUserLocationHandler)
	mux.HandleFunc("POST /user/search", searchUserLocationHandler)

	httpServer = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("started http server at http://localhost:8080")

	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("http server failed to start: %v\n", err)
	}
}

// disconnectMongoClient disconnects the mongo client from the database
func disconnectMongoClient(wg *sync.WaitGroup) {
	defer wg.Done()

	if mongoClient == nil {
		log.Println("mongo client is nil, skipping disconnection")
		return
	}

	log.Println("disconnecting mongo client...")

	if err := mongoClient.Disconnect(); err != nil {
		log.Printf("error disconnecting mongo client: %v\n", err)

	} else {
		log.Println("successfully disconnected mongo client")
	}
}

// disconnectLocationHistoryManagementClient disconnects the location history management client
func disconnectLocationHistoryManagementClient(wg *sync.WaitGroup) {
	defer wg.Done()

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

// shutdownHttpServer shuts down the HTTP server gracefully
// if it does not shutdown in 10 seconds, it will force shutdown
func shutdownHttpServer(wg *sync.WaitGroup) {
	defer wg.Done()

	if httpServer == nil {
		log.Println("http server is nil, skipping shutdown")
		return
	}

	log.Println("shutting down http server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("error shutting down http server: %v\n", err)

	} else {
		log.Println("successfully shut down http server")
	}
}
