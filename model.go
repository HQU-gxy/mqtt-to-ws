package main

import (
	"context"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Ctx = context.TODO()
)

const (
	recordPerPage = 10
)

type MQTTMsg struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

func (m *MQTTMsg) ToRecord() (MQTTRecord, error) {
	payload, err := strconv.ParseFloat(m.Payload, 32)
	return MQTTRecord{
		Payload:   payload,
		Timestamp: time.Now(),
	}, err
}

type MQTTRecord struct {
	Payload   float64   `bson:"payload" json:"payload"`
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
}

func GetDB(uri string, db string) (*mongo.Database, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(Ctx, clientOptions)
	if err != nil {
		lsugar.Error(err)
		return nil, err
	}

	err = client.Ping(Ctx, nil)
	if err != nil {
		lsugar.Error(err)
		return nil, err
	}

	return client.Database(db), err
}

func CreateRecord(db *mongo.Database, collection string, data MQTTRecord) error {
	_, err := db.Collection(collection).InsertOne(Ctx, data)
	if err != nil {
		lsugar.Error(err)
	}

	return err
}

func GetRecords(db *mongo.Database, collection string, filter interface{}, opts *options.FindOptions) ([]MQTTRecord, error) {
	cur, err := db.Collection(collection).Find(Ctx, filter, opts)
	if err != nil {
		lsugar.Error(err)
		return nil, err
	}

	var results []MQTTRecord
	for cur.Next(Ctx) {
		var result MQTTRecord
		err := cur.Decode(&result)
		if err != nil {
			lsugar.Error(err)
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

func GetOptions(page int64) *options.FindOptions {
	opts := options.Find()
	opts.SetLimit(recordPerPage)
	opts.SetSkip(recordPerPage * (page - 1))
	opts.SetSort(bson.M{"timestamp": -1})
	return opts
}

func GetRecordsByPage(db *mongo.Database, collection string, page int64) ([]MQTTRecord, error) {
	opts := GetOptions(page)
	// filter should not be nil
	return GetRecords(db, collection, bson.D{}, opts)
}

func GetRecordsFrom(db *mongo.Database, collection string, start time.Time, page int64) ([]MQTTRecord, error) {
	opts := GetOptions(page)
	// https://stackoverflow.com/questions/54548441/composite-literal-uses-unkeyed-fields
	filter := bson.D{
		{"timestamp", bson.D{
			{"$gte", start},
		}},
	}
	return GetRecords(db, collection, filter, opts)
}

func GetRecordsBetween(db *mongo.Database, collection string, start time.Time, end time.Time, page int64) ([]MQTTRecord, error) {
	opts := GetOptions(page)
	// https://stackoverflow.com/questions/54548441/composite-literal-uses-unkeyed-fields
	filter := bson.D{
		{"timestamp", bson.D{
			{"$gte", start},
			{"$lte", end},
		}},
	}
	return GetRecords(db, collection, filter, opts)
}
