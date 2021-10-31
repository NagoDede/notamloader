package database

import (
	_ "compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"path/filepath"
	"io"
	"github.com/NagoDede/notamloader/notam"
	. "github.com/ahmetb/go-linq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongodb struct {
	client       *mongo.Client
	activeNotams *[]notam.NotamStatus
	CountryCode  string
}

var client *mongo.Client
var notamCollection *mongo.Collection

var ctx = context.TODO()

func NewMongoDb(countrycode string) *Mongodb {
	fmt.Println("Connect to NOTAM database")
	ctx = context.TODO()
	clientmg := getClient()
	mgdb := &Mongodb{client: clientmg, CountryCode: countrycode}
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
			mgdb.SetOperable(notam)
		}
	}
}

func (mgdb *Mongodb) UpdateActiveNotamsFromDb() *[]notam.NotamStatus {
	mgdb.activeNotams = mgdb.retrieveActiveNotams()
	fmt.Printf("\t Retrieve %d active NOTAM in the database \n", len(*mgdb.activeNotams))
	return mgdb.activeNotams
}

// Retrieve the Operable Notams in the database
func (mgdb *Mongodb) GetActiveNotamsData() *[]notam.Notam {
	filter := bson.D{{"status", "Operable"}, {"notamreference.countrycode", mgdb.CountryCode}}
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
func (mgdb *Mongodb) WriteActiveNotamToFile(path string) {
	// Getting absolute path of hello.go
	abs, err1 := filepath.Abs(path)
	// Printing if there is no error
	if err1 == nil {
		fmt.Println("Absolute path is:", abs)
	} else {
	log.Fatal(err1)
	}
	
	
	name := filepath.Base(path)
	tmpFile := filepath.Join(os.TempDir(), name)
 
	// Getting absolute path of hello.go
	abs, err1 = filepath.Abs(path)
	// Printing if there is no error
	if err1 == nil {
		fmt.Println("Absolute path is:", abs)
	} else {
	log.Fatal(err1)
	}


	var notamToPrint = mgdb.GetActiveNotamsData()
	fmt.Printf("Notams to print: %i \n", len(*notamToPrint))
	
	os.Remove(tmpFile)

	file, err := os.OpenFile(tmpFile, os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	//writer := gzip.NewWriter(file)
	//defer writer.Close()
	encoder := json.NewEncoder(file)
	defer file.Close()
	encoder.Encode(notamToPrint)

	formatNotamFile(tmpFile)
	
	fi, err := os.Stat(tmpFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("The file is %d bytes long", fi.Size())
	
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
        _, err := io.Copy(destination, source)
	
	fi, err = os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("The copied file is %d bytes long", fi.Size())
	
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
func (mgdb *Mongodb) retrieveActiveNotams() *[]notam.NotamStatus {
	filter := bson.D{{"status", "Operable"}, {"notamreference.countrycode", mgdb.CountryCode}}
	projection := bson.D{
		{"notamreference", 1},
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

func (mgdb Mongodb) IsOldNotam(key string) bool {
	//	IsOldNotam(notam_location string, notam_number string)
	if *mgdb.activeNotams == nil {
		return false
	}

	for _, ntmref := range *mgdb.activeNotams {
		if ntmref.GetKey() == key {
			return true
		}
	}
	return false
}

func (mgdb Mongodb) IdentifyCanceledNotams(currentNotams map[string]notam.NotamReference) *[]notam.NotamStatus {
	var canceledNotams []notam.NotamStatus

	From(*mgdb.activeNotams).Where(func(c interface{}) bool {
		for _, ntm := range currentNotams {
			// if ntm.Icaolocation == c.(notam.NotamStatus).Icaolocation &&
			// 	ntm.Number == c.(notam.NotamStatus).Number {
			// 	return false
			// }
			var ntmRef notam.NotamStatus
			ntmRef = c.(notam.NotamStatus)
			if ntm.GetKey() == ntmRef.GetKey() {
				return false
			}
		}
		return true
	}).ToSlice(&canceledNotams)

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
