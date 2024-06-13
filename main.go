package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"reflect"

	"github.com/mitchellh/mapstructure"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/slices"

	"alertmanager/models"
	"alertmanager/ruleengine"
	"alertmanager/utilities"
)

const mongouri = "mongodb://localhost:27017/api"
const mongodatabase = "myapp_development"
const mongocollection = "alerts"


// Wrapper type around models.CustomTime
type CustomTimeWrapper struct {
    models.CustomTime
}

func (ct *CustomTimeWrapper) UnmarshalJSON(b []byte) error {
    s := strings.Trim(string(b), `"`) // Remove quotes from the string
	if len(b) < 3 {
		ct.Time = time.Time{}
		return nil
	}
    t, err := time.Parse("2006-01-02 15:04:05", s)
    if err != nil {
        return err
    }
    ct.Time = t
    return nil
}

func check_flap() (string,error){
	var rulesGroup ruleengine.RulesGroup

	rule := `{
		"condition": "AND",
		"rules": [
		  {
			"id": "name",
			"field": "name",
			"type": "string",
			"input": "text",
			"operator": "contains",
			"value": "John"
		  }
		]
	  }`

	err := json.Unmarshal([]byte(rule), &rulesGroup)
	if err != nil {
		fmt.Println("Error in rule evaluation ", err)
	}

	data := map[string]interface{}{
		"name":      "John",
		"age":       30,
		"birthdate": time.Date(1990, 6, 12, 0, 0, 0, 0, time.UTC),
	}

	res := ruleengine.EvaluateRulesGroup(data, rulesGroup)

	fmt.Println("The result is ", res)

	return "OK", nil
}

func main() {
	
	fmt.Println("\n\x1b[32mStarting EA API Server.....\x1b[0m\n")
	fmt.Println("\x1b[32mStarting mongo connection.....\x1b[0m\n")

 	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(mongouri).SetServerAPIOptions(serverAPI)
	// Create a new mongoClient and connect to the server
	mongoClient, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = mongoClient.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	// Send a ping to confirm a successful connection
	var result bson.M
	if err := mongoClient.Database("admin").RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}).Decode(&result); err != nil {
		panic(err)
	}
	fmt.Println("\x1b[32mPinged your deployment. You successfully connected to MongoDB!\x1b[0m\n ")
	fmt.Println("\x1b[32mWaiting for alerts.....\x1b[0m\n")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		Handler(w, r, mongoClient  )
	 })
	http.ListenAndServe(":8081", nil)
}

func Handler(w http.ResponseWriter, r *http.Request, mongoClient *mongo.Client ) {

	alertCollection := mongoClient.Database(mongodatabase).Collection(mongocollection)

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var apiAlertData utilities.ApiAlertData

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	if err := json.Unmarshal(body, &apiAlertData); err != nil {
		fmt.Println(err)
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	// check if any of the required fields are empty.
	if key, err := apiAlertData.IsEmpty("entity", "alertTime", "alertSource", "serviceName", "alertSummary", "severity", "alertId"); err != nil {
		fmt.Printf("The key '%s' is missing or empty: %s\n", key, err)
		return
	} else {
		fmt.Println("All keys are present and not empty")
	}

	if apiAlertData["alertType"] == "CREATE"{
		fmt.Println("This is a create event")
		//status , err := check_flap()
		//fmt.Println("Here", status , err)
		fmt.Println("The API alertId is ", apiAlertData["alertId"])

		// De-duplication Starts
		filter := bson.M{
			"alertid":     apiAlertData["alertId"].(string) ,
		}

		var existingEvent models.DbAlert
		existingEvent = models.DbAlert{}
		
		opts := options.FindOne()
		err1 := alertCollection.FindOne(context.TODO(), filter, opts).Decode(&existingEvent)
	
		if err1 != nil {
			if err1 == mongo.ErrNoDocuments {
				fmt.Println("No matching event found. Creating Alert....")
				layout := "2006-01-02 15:04:05"
				parsedTime, err := time.Parse(layout, apiAlertData["alertTime"].(string))
				if err != nil {
					fmt.Println("Error parsing time:", err)
					return
				}
				// Create a alert in DB
				newAlert := models.DbAlert{
					Entity:				apiAlertData["entity"].(string),
					AlertFirstTime:		models.CustomTime{Time: parsedTime},
					AlertLastTime:		models.CustomTime{},
					AlertClearTime:		models.CustomTime{},
					AlertSource:		apiAlertData["alertSource"].(string),
					ServiceName: 		apiAlertData["serviceName"].(string),
					AlertSummary:		apiAlertData["alertSummary"].(string),
					AlertStatus:		"OPEN",
					AlertNotes:			apiAlertData["alertNotes"].(string),
					AlertAcked:			"NO",
					Severity:			apiAlertData["severity"].(string),
					AlertId:			apiAlertData["alertId"].(string),
					AlertPriority:		"NORMAL",
					IpAddress:			apiAlertData["ipAddress"].(string),
					AlertCount:			1,
				}


				// Add additional Tags
				//fmt.Println("The object before addTags is " , newAlert )
				addTags(apiAlertData, &newAlert)
				//fmt.Println("The object after addTags is " , newAlert )
				processAlertRules( &newAlert , mongoClient)

				_, inserterr := alertCollection.InsertOne(context.TODO(), newAlert)

				if inserterr != nil {
					fmt.Println("Insert Error")
					log.Fatal(inserterr)
				}

				alertjsonData, err := json.Marshal(newAlert)
				if err != nil {
					fmt.Println("Error marshaling JSON:", err)
					return
				}
				
				w.Header().Add("Content-Type" , "application/json")
				w.WriteHeader(201)
				w.Write([]byte(alertjsonData))
				fmt.Println("Inserted document successfully")
			} else {
				// Some other fatal error
				log.Fatal(err)
			}
		
		} else {
			// Duplicate Alert
			fmt.Printf("Found event: %+v\n", existingEvent)
			updatefilter := bson.M{"_id": existingEvent.ID }

			update := bson.M{
				"$set": bson.M{
					"alertcount": existingEvent.AlertCount + 1 ,
				},
			}
		
			updateResult , updateerr := alertCollection.UpdateOne(context.TODO(), updatefilter, update)
			if updateerr != nil {
				panic(err)
			}
			if updateResult.ModifiedCount > 0 {
				fmt.Printf("Matched %v documents and updated %v documents.\n", updateResult.MatchedCount, updateResult.ModifiedCount)
			}
			alertjsonData, err := json.Marshal(apiAlertData)
			if err != nil {
				fmt.Println("Error marshaling JSON:", err)
				return
			}
			w.WriteHeader(200)
			w.Write([]byte(alertjsonData))
			
		}

	// De-duplication Ends
	}else{
		fmt.Println("This is a close event")
	}
}

func addTags(apiAlertData map[string]interface{}, newAlert *models.DbAlert) bool {
	Tags := make(map[string]interface{})
	exA := utilities.ExcludeAttributes{
		AlertTagExclude: []string{"entity","alertTime", "alertNotes", "severity","alertId","ipAddress","alertType","serviceName","alertSummary"},
	}

	for alertJsonKey, alertJsonValue  := range apiAlertData {

		if slices.Contains(exA.AlertTagExclude, alertJsonKey){
			// fmt.Printf(" %v is in exclude attribute\n",alertJsonKey )
		}else{
			// fmt.Printf(" %v is NOT in exclude attribute\n",alertJsonKey )
			// Add to the new map
			Tags[alertJsonKey] = alertJsonValue
		}
	}
	newAlert.AdditionalDetails = Tags 
	//fmt.Println("The object in addTags is " , newAlert )
	return true
}

func processAlertRules(newAlert *models.DbAlert, mongoClient *mongo.Client) bool {
	var rulesGroup ruleengine.RulesGroup
	alertRulesCollection := mongoClient.Database(mongodatabase).Collection("alertrules")

	cursor, err := alertRulesCollection.Find(context.TODO(), bson.D{})
    if err != nil {
        log.Fatal(err)
    }
    defer cursor.Close(context.TODO())

	var alertRules []models.DbAlertRule
	if err = cursor.All(context.TODO(), &alertRules); err != nil {
        log.Fatal(err)
    }

	for _, alertRule   := range alertRules {
		// fmt.Println("Rule is ", alertRule.RuleObject)
		err := json.Unmarshal([]byte(alertRule.RuleObject), &rulesGroup)
		if err != nil {
			fmt.Println("Error in rule evaluation ", err)
		}
		var alertMap map[string]interface{}
		err1 := mapstructure.Decode(newAlert, &alertMap)
		if err1 != nil {
			fmt.Println("ERROR : Unable to convert struct to map")
		}
		//fmt.Println("THE ALERT MAP IS ", alertMap)
		res := ruleengine.EvaluateRulesGroup(alertMap, rulesGroup)
	
		if res {
			// Do the action specified in the rule
			v := reflect.ValueOf(newAlert).Elem()
			field := v.FieldByName(alertRule.SetField)
			if !field.IsValid() ||  !field.CanSet()  {
				fmt.Println("ERROR : The struct element is un settable")
				continue
			}
			fieldValue := reflect.ValueOf(alertRule.SetValue)
			field.Set(fieldValue)
		}
		//fmt.Println("The MATCH is ", res)
	}
	return true
}

func IsValidJSON(str string) bool {
    var js json.RawMessage
    return json.Unmarshal([]byte(str), &js) == nil
}

