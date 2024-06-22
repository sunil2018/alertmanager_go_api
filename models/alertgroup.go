package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DbAlertGroup struct {
	ID 					primitive.ObjectID 	`bson:"_id,omitempty"`
	GroupName			string 				`bson:"groupname" json:"groupname"`
	GroupTags 			[]string 			`bson:"grouptags" json:"grouptags"`
	GroupWindow			int  				`bson:"groupwindow" json:"groupwindow"`
}