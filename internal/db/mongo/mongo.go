package mongo

import (
	"context"
	"fmt"
	"os"

	"github.com/b-j-roberts/foc-engine/internal/config"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoDB struct {
	Client *mongo.Client
}

var Mongo *MongoDB

func ShouldConnectMongo() bool {
	if config.ModuleEnabled(config.ModuleAccounts) ||
		config.ModuleEnabled(config.ModuleEvents) ||
		config.ModuleEnabled(config.ModuleRegistry) {
		return true
	}
	return false
}

func InitMongoDB() {
	mongoUri := os.Getenv("MONGO_URI")
	if mongoUri == "" {
		fmt.Println("MONGO_URI env var not set")
		os.Exit(1)
	}
	mongoClient, err := mongo.Connect(options.Client().ApplyURI(mongoUri))
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		os.Exit(1)
	}
	Mongo = &MongoDB{
		Client: mongoClient,
	}
	fmt.Println("Connected to MongoDB")
}

func InsertJson(dbName string, collectionName string, data interface{}) (*mongo.InsertOneResult, error) {
	/*
	  bsonData, err := JsonToBson(data)
	  if err != nil {
	    return nil, fmt.Errorf("Error converting JSON to BSON: %v", err)
	  }
	*/
	collection := Mongo.Client.Database(dbName).Collection(collectionName)
	if collection == nil {
		return nil, fmt.Errorf("Collection not found: %s %s", dbName, collectionName)
	}
	res, err := collection.InsertOne(context.TODO(), data)
	if err != nil {
		return nil, fmt.Errorf("Error inserting document: %v", err)
	}
	return res, nil
}

func JsonToBson(data interface{}) (interface{}, error) {
	bsonData, err := bson.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Error converting JSON to BSON: %v", err)
	}
	return bsonData, nil
}

func GetFocEngineRegistryCollection() *mongo.Collection {
	collection := Mongo.Client.Database("foc_engine").Collection("registry")
	if collection == nil {
		fmt.Println("Collection not found: foc_engine registry")
	}
	return collection
}

func GetFocEngineEventsCollection() *mongo.Collection {
	collection := Mongo.Client.Database("foc_engine").Collection("events")
	if collection == nil {
		fmt.Println("Collection not found: foc_engine events")
	}
	return collection
}

func GetDeviceChallengesCollection() *mongo.Collection {
	collection := Mongo.Client.Database("foc_engine").Collection("device_challenges")
	if collection == nil {
		fmt.Println("Collection not found: foc_engine device_challenges")
	}
	return collection
}
