package main

import (
	context "context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/mmilosevicgd/location-tracking/db"
	pb "github.com/mmilosevicgd/location-tracking/location-history-management/proto"
	"github.com/mmilosevicgd/location-tracking/validation"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

type protoServer struct {
	pb.UnimplementedLocationHistoryManagementServer
}

const (
	locationHistoryCollection = "location-history"
)

var (
	validate    = validator.New()
	mongoClient db.DBClient
	httpServer  *http.Server
	grpcServer  *grpc.Server
)

func main() {
	go initValidations()
	go initMongoClient()
	go initHttpServer()
	go initGrpcServer()

	shutdown, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-shutdown.Done()

	wg := sync.WaitGroup{}
	wg.Add(3)
	go disconnectMongoClient(&wg)
	go shutdownHttpServer(&wg)
	go shutdownGrpcServer(&wg)
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

	mongoClient.MustCreateCollection(locationHistoryCollection)
	mongoClient.MustCreateIndex(locationHistoryCollection, "username", 1)
	mongoClient.MustCreateIndex(locationHistoryCollection, "timestamp", -1)
	mongoClient.MustCreate2dSphereIndex(locationHistoryCollection, "location")
	log.Println("successfully initialized mongo client and created collections and indexes")
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
	mux.HandleFunc("POST /user/distance", calculateUserDistanceHandler)

	httpServer = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("started http server at http://localhost:8080")

	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("http server failed to start: %v\n", err)
	}
}

// initGrpcServer initializes the grpc server and registers the service
func initGrpcServer() {
	if grpcServer != nil {
		log.Println("grpc server already initialized")
		return
	}

	log.Println("initializing grpc server...")
	listener, err := net.Listen("tcp", ":50051")

	if err != nil {
		log.Fatalf("failed to create tcp listener on port 50051: %v\n", err)
	}

	grpcServer = grpc.NewServer()
	pb.RegisterLocationHistoryManagementServer(grpcServer, &protoServer{})
	log.Println("started grpc server at http://localhost:50051")

	if err = grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpc server failed to start: %v\n", err)
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

// shutdownGrpcServer shuts down the grpc server gracefully
// if it does not shutdown in 10 seconds, it will force shutdown
func shutdownGrpcServer(wg *sync.WaitGroup) {
	defer wg.Done()

	if grpcServer == nil {
		log.Println("grpc server is nil, skipping shutdown")
		return
	}

	log.Println("shutting down grpc server...")

	timer := time.AfterFunc(10*time.Second, func() {
		log.Println("grpc server did not shutdown gracefully in time, forcing shutdown...")
		grpcServer.Stop()
	})

	defer timer.Stop()

	grpcServer.GracefulStop()
	log.Println("successfully shut down grpc server")
}
