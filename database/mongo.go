package database

import (
	_ "compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/NagoDede/notamloader/notam"
	_ "github.com/ahmetb/go-linq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongodb struct {
	client       *mongo.Client
	ActiveNotams map[string]*notam.NotamStatus//*[]notam.NotamStatus
	AfsCode  string
}

var client *mongo.Client
var notamCollection *mongo.Collection

var ctx = context.TODO()

func NewMongoDb(afs string) *Mongodb {
	fmt.Println("Connect to NOTAM database")
	ctx = context.TODO()
	clientmg := getClient()
	mgdb := &Mongodb{client: clientmg, AfsCode: afs}
	mgdb.ActiveNotams = make(map[string]*notam.NotamStatus)
	mgdb.UpdateActiveNotamsFromDb()
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
	//check before add that the doc does not exist in
	_, err := notamCollection.InsertOne(ctx, notam)
	if err != nil {
		var merr mongo.WriteException
		merr = err.(mongo.WriteException)
		errCode := merr.WriteErrors[0].Code
		//discard case where key is in database
		if errCode != 11000 {
			log.Fatal(err)
		} else {
			fmt.Printf("NOTAM: %s in database \n", notam.Id)
			if notam.Status != "Operable" {
				mgdb.SetOperable(notam)
			}
		}
	}
}

func (mgdb *Mongodb) UpdateActiveNotamsFromDb() map[string]*notam.NotamStatus {
	mgdb.ActiveNotams = mgdb.retrieveActiveNotams()
	fmt.Printf("\t Retrieve %d active NOTAM in the database \n", len(mgdb.ActiveNotams))
	return mgdb.ActiveNotams
}

// Retrieve the Operable Notams in the database
func (mgdb *Mongodb) GetActiveNotamsData() *[]notam.Notam {
	filter := bson.D{{"status", "Operable"}, {"notamreference.afscode", mgdb.AfsCode}}
	myCursor, err := notamCollection.Find(ctx, filter)
	if err != nil {
		log.Fatal(err)
	}

	var notams []notam.Notam
	if err = myCursor.All(context.Background(), &notams); err != nil {
		log.Fatal(err)
	}
	return &notams
}

// Write all the Active Notams in the indicated file.
// The file is Gzipped.
func (mgdb *Mongodb) WriteActiveNotamToFile(country string, afs string) {
	const dir = "./web/notams/"
	path := dir + country + "_" + afs + ".json"
	
	abs, err1 := filepath.Abs(path)
	// Printing if there is no error
	if err1 == nil {
		fmt.Println("Absolute path is:", abs)
	} else {
		log.Fatal(err1)
	}

	name := filepath.Base(path)
	tmpFile := filepath.Join(os.TempDir(), name)

	abs, err1 = filepath.Abs(tmpFile)
	// Printing if there is no error
	if err1 == nil {
		fmt.Println("Absolute path is:", abs)
	} else {
		log.Fatal(err1)
	}

	var notamToPrint = mgdb.GetActiveNotamsData()
	fmt.Printf("Notams to print: %i \n", len(*notamToPrint))

	os.Remove(tmpFile)

	file, err := os.OpenFile(tmpFile, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	//writer := gzip.NewWriter(file)
	//defer writer.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(notamToPrint)
	if err != nil {
		log.Fatal(err)
	}
	file.Close()

	formatNotamFile(tmpFile)

	fi, err := os.Stat(tmpFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("The file is %d bytes long \n", fi.Size())

	source, err := os.Open(tmpFile)
	if err != nil {
		log.Fatal(err)
	}
	defer source.Close()

	destination, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)

	fi, err = os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("The copied file is %d bytes long \n", fi.Size())

}

func formatNotamFile(path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	text := string(content)
	text = strings.ReplaceAll(text, "},", "},\n")
	text = strings.ReplaceAll(text, "\",\"", "\",\n\"")

	os.Remove(path)
	fdb := os.WriteFile(path, []byte(text), 0666)
	if fdb != nil {
		log.Fatal(fdb)
	}
	fmt.Printf("Success, write File %s \n", path)
}

//
func (mgdb *Mongodb) retrieveActiveNotams() map[string]*notam.NotamStatus{
	filter := bson.D{{"status", "Operable"}, {"notamreference.afscode", mgdb.AfsCode}}
	projection := bson.D{
		{"notamreference", 1},
		{"status", 1},
	}

	myCursor, err := notamCollection.Find(ctx, filter, options.Find().SetProjection(projection))
	if err != nil {
		log.Fatal(err)
	}

	//Retrive the data from the databse.
	//Use a structure to catch the Id and other references
	type localNotam struct{
		Id string `bson:"_id,omitempty"`
		notam.NotamReference
		Status string	`json:"status"`
	}

	var notams = make(map[string]*notam.NotamStatus)//[]notam.NotamStatus
	info := &localNotam{}
	for myCursor.Next(context.Background()) {
		myCursor.Decode(info)
		notams[info.Id] = &(notam.NotamStatus{NotamReference: info.NotamReference, Status: info.Status})
	}

	// if err = myCursor.All(context.Background(), &notams); err != nil {
	//  	log.Fatal(err)
	// }
	return notams
}

func (mgdb Mongodb) IsOldNotam(key string) bool {
	//	IsOldNotam(notam_location string, notam_number string)
	if mgdb.ActiveNotams == nil {
		return false
	}

	for _, ntmref := range mgdb.ActiveNotams {

		if ntmref.GetKey() == key {
			return true
		}
	}
	return false
}

func (mgdb Mongodb) IdentifyCanceledNotams(currentNotams map[string]notam.NotamReference) *[]notam.NotamStatus {
	var canceledNotams []notam.NotamStatus

	for activeKey, active := range mgdb.ActiveNotams{
		_, inCurrent := currentNotams[activeKey]
		if !inCurrent {
			canceledNotams = append(canceledNotams, *active)
		}
	}


	//	From(mgdb.ActiveNotams).Where(func(c interface{}) bool {
	// 	for _, ntm := range currentNotams {
	// 		// if ntm.Icaolocation == c.(notam.NotamStatus).Icaolocation &&
	// 		// 	ntm.Number == c.(notam.NotamStatus).Number {
	// 		// 	return false
	// 		// }
	// 		var ntmRef notam.NotamStatus
	// 		ntmRef = c.(notam.NotamStatus)
	// 		if ntm.GetKey() == ntmRef.GetKey() {
	// 			return false
	// 		}
	// 	}
	// 	return true
	// }).ToSlice(&canceledNotams)

	return &canceledNotams
}

func (mgdb Mongodb) SetOperable(notam *notam.Notam) {
	filter := bson.M{"_id": notam.GetKey()}
	setOperable := bson.D{
		{"$set", bson.D{{"status", "Operable"}}},
	}
	_, err := notamCollection.UpdateOne(ctx, filter, setOperable)
	if err == nil {
		fmt.Printf("%s changed to Operable  \n", notam.Id)
	} else {
		fmt.Printf("Error during change to Operable %s \n", err)
	}
}

func (mgdb Mongodb) SetCanceledNotamList(canceledNotams *[]notam.NotamStatus) {
	if len(*canceledNotams) > 0 {
		for _, canceled := range *canceledNotams {
			filter := bson.M{"status": "Operable",
				"notamreference.number":       canceled.Number,
				"notamreference.icaolocation": canceled.Icaolocation}

			setCancel := bson.D{
				{"$set", bson.D{{"status", "Canceled"}}},
			}
			notamCollection.UpdateMany(ctx, filter, setCancel)
		}
	}
}
