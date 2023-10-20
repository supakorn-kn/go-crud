package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/supakorn-kn/go-crud/env"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBConn struct {
	databaseName string

	client *mongo.Client
	opts   *options.ClientOptions
}

func (db *MongoDBConn) Connect() error {

	client, err := mongo.Connect(context.Background(), db.opts)
	if err != nil {
		return err
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		return err
	}

	db.client = client

	return nil
}

func (db *MongoDBConn) Disconnect() error {
	return db.client.Disconnect(context.Background())
}

func (db *MongoDBConn) GetDatabase() *mongo.Database {
	return db.client.Database(db.databaseName)
}

func (db *MongoDBConn) GetCollection(collectionName string) *mongo.Collection {

	return db.GetDatabase().Collection(collectionName)
}

func New(config env.MongoDBConfig) (*MongoDBConn, error) {

	if config.DB == "" {
		return nil, errors.New("mongoDB database name is required")
	}

	opts, err := createClientOptions(config)
	if err != nil {
		return nil, err
	}

	mongoDBConn := MongoDBConn{
		opts:         opts,
		databaseName: config.DB,
	}

	return &mongoDBConn, nil
}

func createClientOptions(config env.MongoDBConfig) (*options.ClientOptions, error) {

	if config.Host == "" {
		return nil, errors.New("mongoDB host is required")
	}

	if config.Port == 0 {
		return nil, errors.New("mongoDB port is required")
	}

	opts := options.Client()
	opts.SetHosts([]string{fmt.Sprintf("%s:%d", config.Host, config.Port)})

	if config.User != "" && config.Password != "" {

		opts.SetAuth(options.Credential{
			Username: config.User,
			Password: config.Password,
		})
	}

	opts.SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1))
	opts.SetTimeout(3 * time.Second)

	return opts, nil
}
