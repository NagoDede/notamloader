package database

import (
	_ "compress/gzip"
	"context"
	"encoding/json"
	"io"

	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/NagoDede/notamloader/notam"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Mongodb struct {
	client       *mongo.Client
	ActiveNotams map[string]*notam.NotamStatus//*[]notam.NotamStatus
	AfsCode  string
	logger zerolog.Logger
}

var client *mongo.Client
var notamCollection *mongo.Collection

var ctx = context.TODO()

var mainlogger zerolog.Logger

func NewMongoDb(afs string) *Mongodb {

	ctx = context.TODO()
	clientmg := getClient()
	mgdb := &Mongodb{client: clientmg, AfsCode: afs}
	mgdb.ActiveNotams = make(map[string]*notam.NotamStatus)
	mainlogger = log.With().Str("Mongo","main").Logger()
	mgdb.logger = log.With().Str("Mongo - AFS", afs).Logger()
	mgdb.logger.Info().Msg("Connect to NOTAM database")
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
			mainlogger.Fatal().Err(err)
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
			mgdb.logger.Fatal().Err(err)
		} else {
			//mgdb.logger.Trace().Msgf("NOTAM: %s in database", notam.Id)
			if notam.Status != "Operable" {
				mgdb.SetOperable(notam)
			}
		}
	}
}

func (mgdb *Mongodb) UpdateActiveNotamsFromDb() map[string]*notam.NotamStatus {
	mgdb.ActiveNotams = mgdb.retrieveActiveNotams()
	mgdb.logger.Info().Msgf("Retrieve %d active NOTAM in the database for %s", len(mgdb.ActiveNotams), mgdb.AfsCode)
	return mgdb.ActiveNotams
}

// Retrieve the Operable Notams in the database
func (mgdb *Mongodb) GetActiveNotamsData() *[]notam.Notam {
	filter := bson.D{{"status", "Operable"}, {"notamreference.afscode", mgdb.AfsCode}}
	myCursor, err := notamCollection.Find(ctx, filter)
	if err != nil {
		mgdb.logger.Fatal().Err(err)
	}

	var notams []notam.Notam
	if err = myCursor.All(context.Background(), &notams); err != nil {
		mgdb.logger.Fatal().Err(err)
	}
	return &notams
}

// Write all the Active Notams in the indicated file.
// The file is Gzipped.
func (mgdb Mongodb) WriteActiveNotamToFile(country string, afs string) {
	const dir = "./web/notams/"
	path := dir + country + "_" + afs + ".json"

	//Write first in a temporary dir
	name := filepath.Base(path)
	tmpPath := filepath.Join(os.TempDir(), name)

	var notamToPrint = mgdb.GetActiveNotamsData()
	mgdb.logger.Info().Msgf("Notams to print: %d", len(*notamToPrint))

	os.Remove(tmpPath)

	tmpfile, err := os.OpenFile(tmpPath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		mgdb.logger.Fatal().Err(err)
	}

	//writer := gzip.NewWriter(file)
	//defer writer.Close()
	encoder := json.NewEncoder(tmpfile)
	err = encoder.Encode(notamToPrint)
	if err != nil {
		mgdb.logger.Fatal().Err(err)
	}
	tmpfile.Close()

	//The content of the file is not formatted
	//Git does not support well this and file is not easy to read
	formatNotamFile(tmpPath)
	fi, err := os.Stat(tmpPath)
	if err != nil {
		mgdb.logger.Fatal().Err(err)
	}
	//Copy the file
	source, err := os.Open(tmpPath)
	if err != nil {
		mgdb.logger.Fatal().Err(err)
	}
	defer source.Close()

	destination, err := os.Create(path)
	if err != nil {
		mgdb.logger.Fatal().Err(err)
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)

	fi, err = os.Stat(path)
	if err != nil {
		mgdb.logger.Fatal().Err(err)
	}

	mgdb.logger.Info().Msgf("Output file: %s (%d bytes)", path, fi.Size())

	abs, err1 := filepath.Abs(path)
	if err1 != nil {
		mgdb.logger.Fatal().Err(err1)
	}
	mgdb.logger.Trace().Msgf("Absolute path: %s", abs)
}

func formatNotamFile(path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		mainlogger.Fatal().Err(err)
	}

	text := string(content)
	text = strings.ReplaceAll(text, "},", "},\n")
	text = strings.ReplaceAll(text, "\",\"", "\",\n\"")

	os.Remove(path)
	fdb := os.WriteFile(path, []byte(text), 0666)
	if fdb != nil {
		mainlogger.Fatal().Err(fdb)
	}
	mainlogger.Trace().Msgf("Success format file %s", path)
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
		mainlogger.Fatal().Err(err)
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

	return notams
}

func (mgdb Mongodb) IsOldNotam(key string) bool {

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

	return &canceledNotams
}

func (mgdb Mongodb) SetOperable(notam *notam.Notam) {
	filter := bson.M{"_id": notam.GetKey()}
	setOperable := bson.D{
		{"$set", bson.D{{"status", "Operable"}}},
	}
	_, err := notamCollection.UpdateOne(ctx, filter, setOperable)
	if err == nil {
		mgdb.logger.Info().Msgf("%s changed to Operable  \n", notam.Id)
	} else {
		mgdb.logger.Warn().Msgf("Error during change to Operable %s \n", err)
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

func (mgdb Mongodb ) SendToDatabase(fl *notam.NotamList) *notam.NotamReferenceList {

	notamList := notam.NewNotamReferenceList()
	for _, frNotam := range fl.Data {
		frNotam.Status = "Operable"

		//avoid duplicate
		_, ok := notamList.Data[frNotam.Id]
		if !ok {
			//record all notams, except the duplicate
			notamList.Data[frNotam.Id] = frNotam.NotamReference

			//send to db only if necessary
			_, isOld := mgdb.ActiveNotams[frNotam.Id]
			if !isOld {
				mgdb.AddNotam(frNotam)
			}
		} else {
			mgdb.logger.Info().Msgf("Duplicated %s \n", frNotam.Id)
		}
	}
	return notamList
}
