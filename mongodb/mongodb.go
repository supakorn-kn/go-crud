package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBConn struct {
	databaseName string

	client *mongo.Client
	opts   *options.ClientOptions
}

func (db *MongoDBConn) Connect() error {

	client, err := mongo.Connect(context.TODO(), db.opts)
	if err != nil {
		return err
	}

	db.client = client

	return nil
}

func (db *MongoDBConn) Disconnect() error {
	return db.client.Disconnect(context.TODO())
}

func (db *MongoDBConn) GetDatabase() *mongo.Database {
	return db.client.Database(db.databaseName)
}

func (db *MongoDBConn) GetCollection(collectionName string) *mongo.Collection {

	return db.GetDatabase().Collection(collectionName)
}

func New(uri string, dbName string) MongoDBConn {

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)

	return MongoDBConn{
		opts:         opts,
		databaseName: dbName,
	}
}
