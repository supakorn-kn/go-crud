package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBConn struct {
	Client *mongo.Client
	opts   *options.ClientOptions
}

func (db *MongoDBConn) Connect() error {

	client, err := mongo.Connect(context.TODO(), db.opts)
	if err != nil {
		return err
	}

	db.Client = client

	return nil
}

func (db *MongoDBConn) Disconnect() error {
	return db.Client.Disconnect(context.TODO())
}

func (db *MongoDBConn) GetCollection(databaseName, collectionName string) *mongo.Collection {

	return db.Client.Database(databaseName).Collection(collectionName)
}

func New(uri string) MongoDBConn {

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)

	return MongoDBConn{
		opts: opts,
	}
}

func InitConnection(uri string) (*MongoDBConn, error) {
	mongodbConn := New(uri)
	if err := mongodbConn.Connect(); err != nil {
		return nil, err
	}

	return &mongodbConn, nil
}
