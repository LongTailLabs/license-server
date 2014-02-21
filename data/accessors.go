package data

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

// Consumer
func GetConsumers(db *mgo.Database) []Consumer {
	consumers := make([]Consumer, 0, 10)
	db.C("consumers").Find(nil).All(&consumers)
	return consumers
}

func GetConsumer(db *mgo.Database, id string) (Consumer, error) {
	var consumer Consumer
	err := db.C("consumers").Find(bson.M{"_id": id}).One(&consumer)
	return consumer, err
}

func NewConsumer(db *mgo.Database, c *Consumer) error {
	return db.C("consumers").Insert(c)
}

// Application
func GetApplications(db *mgo.Database) []Application {
	applications := make([]Application, 0, 10)
	db.C("applications").Find(nil).All(&applications)
	return applications
}

func GetApplication(db *mgo.Database, id string) (Application, error) {
	var application Application
	err := db.C("applications").Find(bson.M{"_id": id}).One(&application)
	return application, err
}

func NewApplication(db *mgo.Database, c *Application) error {
	return db.C("applications").Insert(c)
}
