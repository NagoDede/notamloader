package database

import (
	"log"
	"fmt"
	"sync"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"context"
	"github.com/NagoDede/notamloader/notam"
	"go.mongodb.org/mongo-driver/bson"
	. "github.com/ahmetb/go-linq"
)

type Mongodb struct {
	client *mongo.Client
	activeNotams *[]notam.NotamStatus
}

var client *mongo.Client
var notamCollection *mongo.Collection

var ctx = context.TODO()

func NewMongoDb() *Mongodb{
	fmt.Println("Connect to NOTAM database")
	ctx = context.TODO()
	clientmg := getClient()
	mgdb := &Mongodb{client: clientmg,}
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
	fmt.Println(notam)
	_, err := notamCollection.InsertOne(ctx, notam)
	if err != nil {
		log.Fatal(err)
	}
}

func (mgdb *Mongodb) GetActiveNotamsInDb() *[]notam.NotamStatus {
	mgdb.activeNotams = mgdb.retrieveActiveNotams()
	fmt.Printf("\t Retrieve %d active NOTAM in the database \n", len(*mgdb.activeNotams))
	return mgdb.activeNotams
}

//
func (mgdb *Mongodb) retrieveActiveNotams() *[]notam.NotamStatus {
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

	var notams []notam.NotamStatus
	if err = myCursor.All(context.Background(), &notams); err != nil {
		log.Fatal(err)
	}
	return &notams
}

func (mgdb Mongodb) IsOldNotam( notam_location string, notam_number string) bool {

	if *mgdb.activeNotams == nil {
		return false
	}

	for _, ntmref := range *mgdb.activeNotams {
		if ntmref.Icaolocation == notam_location && 
			ntmref.Number == notam_number {
			return true
		}
	}
	return false
}

func (mgdb Mongodb) IdentifyCanceledNotams(currentNotams *[]notam.NotamReference) *[]notam.NotamStatus {
	var canceledNotams []notam.NotamStatus

	From(*mgdb.activeNotams).Where(func(c interface{}) bool {
		for _, ntm := range *currentNotams {
			if ntm.Icaolocation == c.(notam.NotamStatus).Icaolocation && 
				ntm.Number == c.(notam.NotamStatus).Number {
				return false
			}
		}
		return true
	}).ToSlice(&canceledNotams)

		return &canceledNotams
}

func (mgdb Mongodb) SetCanceledNotamList(canceledNotams *[]notam.NotamStatus) {
	if len(*canceledNotams) >0 {
		for _, canceled := range *canceledNotams {
			filter := bson.M{"status": "Operable", 
			"notamreference.number": canceled.Number, 
			"notamreference.icaolocation": canceled.Icaolocation}

			setCancel := bson.D{
				{"$set", bson.D{{"status","Canceled"}}},
			}

			notamCollection.UpdateMany(ctx, filter,setCancel)
		}
	}
}