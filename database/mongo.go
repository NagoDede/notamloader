package database

import (
	"context"

	"log"
	"sync"

	"github.com/NagoDede/notamloader/notam"
	. "github.com/ahmetb/go-linq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongodb struct {
	client       *mongo.Client
	activeNotams []notam.NotamReference
}

var client *mongo.Client
var notamCollection *mongo.Collection

var ctx = context.TODO()

func NewMongoDb() *Mongodb {
	ctx = context.TODO()
	clientmg := getClient()
	mgdb := &Mongodb{client: clientmg}
	mgdb.GetActiveNotamsInDb()
	return mgdb
}

func getClient() *mongo.Client {
	var once sync.Once
	onceBody := func() {

		var err error
		client, err = mongo.Connect(ctx, options.Client().ApplyURI(
			"mongodb+srv://notamuser:notamuser@clusternotam.6y9s1.mongodb.net",
		))
		if err != nil {
			log.Fatal(err)
		}
	}

	once.Do(onceBody)
	notamCollection = client.Database("NOTAMS").Collection("notams")
	return client
}

func (mgdb *Mongodb) AddNotam(notam *notam.Notam) {
	_, err := notamCollection.InsertOne(ctx, notam)
	mgdb.activeNotams = append(mgdb.activeNotams, notam.NotamReference)
	if err != nil {
		log.Fatal(err)
	}
}

func (mgdb *Mongodb) GetActiveNotamsInDb() []notam.NotamReference {
	mgdb.activeNotams = mgdb.retrieveActiveNotams()
	return mgdb.activeNotams
}

//
func (mgdb *Mongodb) retrieveActiveNotams() []notam.NotamReference {
	filter := bson.D{{"status", "Operable"}}
	projection := bson.D{
		{"notamreference.number", 1},
		{"notamreference.icaolocation", 1},
		{"status", 1},
	}

	myCursor, err := notamCollection.Find(ctx, filter, options.Find().SetProjection(projection))
	if err != nil {
		log.Fatal(err)
	}

	var notams []notam.NotamReference

	for myCursor.Next(context.TODO()) {
		var elem notam.Notam
		err := myCursor.Decode(&elem)
		if err != nil {
			log.Fatal(err)
		}
		notams = append(notams, notam.NotamReference{
			Number:       elem.Number,
			Icaolocation: elem.Icaolocation})
	}
	return notams
}

func (mgdb Mongodb) IsNewNotam(ntm notam.NotamReference) bool {
	for _, ntmref := range mgdb.activeNotams {
		if ntmref.Icaolocation == ntm.Icaolocation &&
			ntmref.Number == ntm.Number {
			return true
		}
	}
	return false
}

func (mgdb Mongodb) IdentifyCanceledNotams(currentNotams *[]notam.NotamReference) *[]notam.NotamReference {
	var canceledNotams []notam.NotamReference

	From(mgdb.activeNotams).Where(func(c interface{}) bool {
		for _, ntm := range *currentNotams {
			if ntm.Icaolocation == c.(notam.NotamReference).Icaolocation &&
				ntm.Number == c.(notam.NotamReference).Number {
				return false
			}
		}
		return true
	}).ToSlice(&canceledNotams)

	return &canceledNotams
}

func (mgdb Mongodb) SetCanceledNotamList(canceledNotams *[]notam.NotamReference) {
	if len(*canceledNotams) > 0 {
		for _, canceled := range *canceledNotams {
			filter := bson.M{"status": "Operable",
				"number":       canceled.Number,
				"icaolocation": canceled.Icaolocation}

			setCancel := bson.D{
				{"$set", bson.D{{"status", "Canceled"}}},
			}

			notamCollection.UpdateMany(ctx, filter, setCancel)
		}
	}
}
