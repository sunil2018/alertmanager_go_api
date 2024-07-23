package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DbAlertSource struct {
	ID 						primitive.ObjectID 	`bson:"_id,omitempty" json:"_id"`
	AlertSourceName			string 				`bson:"alertsourcename" json:"alertsourcename"`
	AlertSourceDescription 	string 				`bson:"alertsourcedescription" json:"alertsourcedescription"`	
	AlertSourceTransformer 	string 				`bson:"alertsourcetransformer" json:"alertsourcetransformer"`	
}