package deviceverification

import (
	"context"
	"fmt"
	"time"

	"github.com/b-j-roberts/foc-engine/internal/db/mongo"
	"go.mongodb.org/mongo-driver/v2/bson"
	md "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ChallengeDocument represents a challenge stored in MongoDB
type ChallengeDocument struct {
	AccountAddress string    `bson:"account_address"`
	Challenge      string    `bson:"challenge"`
	Expiry         time.Time `bson:"expiry"`
	CreatedAt      time.Time `bson:"created_at"`
}

// InitChallengeCollection initializes the MongoDB collection with TTL index
func InitChallengeCollection() error {
	collection := mongo.GetDeviceChallengesCollection()

	// Create TTL index on expiry field (15 minutes = 900 seconds)
	indexModel := md.IndexModel{
		Keys: bson.D{
			{Key: "expiry", Value: 1},
		},
		Options: options.Index().SetExpireAfterSeconds(0), // TTL index - documents expire after expiry time
	}

	_, err := collection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		// Index might already exist, which is fine
		return nil
	}

	return nil
}

// SaveChallenge saves a challenge to MongoDB with TTL
func SaveChallenge(accountAddress, challenge string, expiry time.Time) error {
	collection := mongo.GetDeviceChallengesCollection()

	// Delete any existing challenge for this account
	_, err := collection.DeleteMany(context.Background(), bson.M{
		"account_address": accountAddress,
	})
	if err != nil {
		return fmt.Errorf("failed to delete existing challenge: %w", err)
	}

	doc := ChallengeDocument{
		AccountAddress: accountAddress,
		Challenge:      challenge,
		Expiry:         expiry,
		CreatedAt:      time.Now(),
	}

	_, err = collection.InsertOne(context.Background(), doc)
	if err != nil {
		return fmt.Errorf("failed to save challenge: %w", err)
	}

	return nil
}

// GetChallenge retrieves a challenge for an account address
func GetChallenge(accountAddress string) (string, error) {
	collection := mongo.GetDeviceChallengesCollection()

	var doc ChallengeDocument
	err := collection.FindOne(context.Background(), bson.M{
		"account_address": accountAddress,
	}).Decode(&doc)

	if err == md.ErrNoDocuments {
		return "", fmt.Errorf("challenge not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get challenge: %w", err)
	}

	// Check if challenge has expired
	if time.Now().After(doc.Expiry) {
		// Delete expired challenge
		collection.DeleteOne(context.Background(), bson.M{
			"account_address": accountAddress,
		})
		return "", fmt.Errorf("challenge expired")
	}

	return doc.Challenge, nil
}

// DeleteChallenge removes a challenge from MongoDB
func DeleteChallenge(accountAddress string) error {
	collection := mongo.GetDeviceChallengesCollection()

	_, err := collection.DeleteOne(context.Background(), bson.M{
		"account_address": accountAddress,
	})
	if err != nil {
		return fmt.Errorf("failed to delete challenge: %w", err)
	}

	return nil
}
