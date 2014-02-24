/**
* Give Martini a try, it uses dependency injection and some other
* sorts of strange magic. But overall it seems like the wishful
* simplicity that might be good for this project. Unless.
*
* Keeping DB/Intelligence as separated as possible so it won't be
* hard to rip out Martini (the HTTP part) if it becomes an issue.

TODO: Consider using constants for the collection names.

*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/LongTailLabs/license-server/data"
	"github.com/LongTailLabs/license-server/restrictors"
	"github.com/sigu-399/gojsonschema"

	"github.com/codegangsta/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

func main() {
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Use(DB())

	setupHandlers(m)

	http.ListenAndServe("127.0.0.1:3000", m)
}

// Meant to be a middleware. This middleware registers a service that is
// to be used as a dependency injected into a handler.
func DB() martini.Handler {
	session, err := mgo.Dial("mongodb://localhost")
	if err != nil {
		panic(err)
	}

	return func(c martini.Context) {
		s := session.Clone()
		db := s.DB("jca-license-server")
		c.Map(db)

		// Unique Index on restrictions(consumer, application)
		restrictionIndex := mgo.Index{
			Key:    []string{"consumer", "application"},
			Unique: true,
		}
		err := db.C("restrictions").EnsureIndex(restrictionIndex)
		if err != nil {
			panic(err.Error())
		}

		// Unique Index on counters(consumer, application, client)
		counterIndex := mgo.Index{
			Key:    []string{"consumer", "application", "client"},
			Unique: true,
		}
		err = db.C("counters").EnsureIndex(counterIndex)
		if err != nil {
			panic(err.Error())
		}

		defer s.Close()
		c.Next()
	}
}

// Separating, easy to move later.
func setupHandlers(m *martini.ClassicMartini) {
	m.Get("/", func(r render.Render) {
		r.JSON(200, map[string]interface{}{
			"@": "IRATEMONK 0.1.0",
			"links": []map[string]string{
				{"rel": "consumers", "href": "/consumers"},
				{"rel": "applications", "href": "/applications"},
				{"rel": "restrictions", "href": "/restrictions/:consumer/:product"},
			},
		})
	})

	// CONSUMERS CRUD (-UD)
	m.Get("/consumers", listByType("consumers"))
	m.Get("/consumers/:id", findById("consumers"))
	m.Post("/consumers", binding.Bind(data.Consumer{}),
		func(db *mgo.Database, c data.Consumer, r render.Render) {
			genericIdInsert(db, "consumers", c, r)
		})

	// PRODUCTS CRUD (-UD)
	m.Get("/applications", listByType("applications"))
	m.Get("/application/:id", findById("applications"))
	m.Post("/applications", binding.Bind(data.Application{}),
		func(db *mgo.Database, a data.Application, r render.Render) {
			genericIdInsert(db, "applications", a, r)
		})

	// Restrictions
	m.Get("/restrictions/:consumer/:application",
		checkIdParam("consumer", "consumers"),
		checkIdParam("application", "applications"),
		func(db *mgo.Database, params martini.Params, r render.Render) {
			var listing []bson.M
			db.C("restrictions").Find(nil).All(&listing)
			r.JSON(200, listing)
		})
	// Add a restriction
	m.Post("/restrictions/:consumer/:application",
		checkIdParam("consumer", "consumers"),
		checkIdParam("application", "applications"),
		validateJSONPayload("schema/restriction.schema"),
		func(db *mgo.Database, requestJSON JSONObject, params martini.Params, r render.Render) {

			_, err := db.C("restrictions").Upsert(
				restrictionQuerySelector(params),
				bson.M{
					"$addToSet": bson.M{"restrictions": requestJSON.Object},
				},
			)

			if err != nil {
				r.JSON(500, data.APIError{Error: "Adding Restriction failed", Context: err.Error()})
			}
		})
	// Replace all restrictions
	m.Put("/restrictions/:consumer/:application",
		checkIdParam("consumer", "consumers"),
		checkIdParam("application", "applications"),
		validateJSONPayload("schema/restriction.schema"),
		func(db *mgo.Database, requestJSON JSONObject, params martini.Params, r render.Render) {

			_, err := db.C("restrictions").Upsert(
				restrictionQuerySelector(params),
				bson.M{
					"$set": bson.M{"restrictions": []bson.M{requestJSON.Object}},
				},
			)

			if err != nil {
				r.JSON(500, data.APIError{Error: "Adding Restriction failed", Context: err.Error()})
			}
		})
	// Delete a restriction
	m.Delete("/restrictions/:consumer/:application",
		checkIdParam("consumer", "consumers"),
		checkIdParam("application", "applications"),
		validateJSONPayload("schema/restriction.schema"),
		func(db *mgo.Database, requestJSON JSONObject, params martini.Params, r render.Render) {

			_, err := db.C("restrictions").Upsert(
				restrictionQuerySelector(params),
				bson.M{
					"$pull": bson.M{"restrictions": requestJSON.Object},
				},
			)

			if err != nil {
				r.JSON(500, data.APIError{Error: "Removing Restriction failed", Context: err.Error()})
			}
		})

	// COUNTING
	m.Post("/signal/:consumer/:application/:action",
		checkIdParam("consumer", "consumers"),
		checkIdParam("application", "applications"),
		binding.Bind(data.Client{}),
		// One off check for action parameter.
		func(params martini.Params, r render.Render) {
			if params["action"] != "use" && params["action"] != "access" {
				r.JSON(400, data.APIError{Error: "Only the actions 'use' and 'access' are valid."})
				return
			}
		},
		// Validate or Initialize
		func(db *mgo.Database, req *http.Request, params martini.Params, c data.Client, r render.Render) {
			var restriction restrictors.Restriction

			err, counter := getCounterForClient(db, params["consumer"], params["application"], c.Id)

			if err != nil {
				if err == mgo.ErrNotFound {
					// No counts for this client yet... let's go ahead and incrememnt by zero to init the data.
					// Mongo $incr by zero is fucking broke... so zero is a special case, check the incrementCounter fn.
					incrementCounter(db, params["consumer"], params["application"], c.Id, params["action"], 0)
					return
				}
				r.JSON(500, data.APIError{Error: "There was an error validating the signal request", Context: params})
				return
			}

			err = db.C("restrictions").Find(restrictionQuerySelector(params)).One(&restriction)

			if err != nil {
				r.JSON(500, data.APIError{Error: "Finding Restriction failed", Context: err.Error()})
				return
			}

			validationErrors := make([]string, 0)
			validators := restriction.MakeValidators()
			for _, validator := range validators {
				err := validator.Validate(req, counter)
				if err != nil {
					validationErrors = append(validationErrors, err.Error())
				}
			}

			if len(validationErrors) > 0 {
				r.JSON(400, bson.M{
					"accepted": false,
					"errors":   validationErrors,
				})

				// Record this transaction
				db.C("requests").Insert(bson.M{
					"consumer":    params["consumer"],
					"application": params["application"],
					"client":      c.Id,
					"request":     params["action"],
					"accepted":    true,
					"reason":      validationErrors,
					"from":        req.RemoteAddr,
					"when":        time.Now(),
				})

				return
			}

		},
		func(db *mgo.Database, req *http.Request, params martini.Params, c data.Client, r render.Render) {
			// Rollup to consumer+application
			err := incrementCounter(db, params["consumer"], params["application"], "", params["action"], 1)
			if err != nil {
				r.JSON(500, data.APIError{Error: "There was an error incrementing the counter", Context: err.Error()})
				return
			}

			// Normal Submission
			err = incrementCounter(db, params["consumer"], params["application"], c.Id, params["action"], 1)
			if err != nil {
				r.JSON(500, data.APIError{Error: "There was an error incrementing the counter", Context: err.Error()})
				return
			}

			r.JSON(200, bson.M{"accepted": true})

			// Record this transaction
			db.C("requests").Insert(bson.M{
				"consumer":    params["consumer"],
				"application": params["application"],
				"client":      c.Id,
				"request":     params["action"],
				"accepted":    true,
				"from":        req.RemoteAddr,
				"when":        time.Now(),
			})
		},
	)

}

// Move to another file.
// Functions that return other functions (views)
type listFn func(db *mgo.Database, r render.Render)
type lookupFn func(db *mgo.Database, params martini.Params, r render.Render)
type insertFn func(db *mgo.Database, obj interface{}, r render.Render)
type paramCheckFn func(db *mgo.Database, params martini.Params, r render.Render)
type bodyValidateFn func(req *http.Request, c martini.Context, r render.Render)

type JSONObject struct {
	Object map[string]interface{}
}

func listByType(collection string) listFn {
	return func(db *mgo.Database, r render.Render) {
		listing := make([]interface{}, 0, 10)
		db.C("consumers").Find(nil).All(&listing)
		r.JSON(200, listing)
	}
}

func findById(collection string) lookupFn {
	return func(db *mgo.Database, params martini.Params, r render.Render) {
		var obj interface{}

		err := db.C(collection).Find(bson.M{"_id": params["id"]}).One(&obj)

		if err != nil {
			if err == mgo.ErrNotFound {
				r.JSON(404, data.APIError{Error: "Not Found", Context: params})
			} else {
				r.JSON(500, data.APIError{Error: "Server Error", Context: err})
			}
		} else {
			r.JSON(200, obj)
		}
	}
}

func checkIdParam(name string, collection string) paramCheckFn {
	return func(db *mgo.Database, params martini.Params, r render.Render) {
		var obj interface{}
		err := db.C(collection).Find(bson.M{"_id": params[name]}).One(&obj)
		if err != nil {
			if err == mgo.ErrNotFound {
				r.JSON(404, data.APIError{Error: fmt.Sprintf("%s not found in %s", name, collection),
					Context: params})
			} else {
				r.JSON(500, data.APIError{Error: "Server Error", Context: err})
			}
			return
		}
	}
}

func validateJSONPayload(schemaAssetName string) bodyValidateFn {
	return func(req *http.Request, c martini.Context, r render.Render) {
		var err error
		var schemaObject map[string]interface{}
		var requestBytes []byte
		var requestObject map[string]interface{}

		// Read request body into bytes
		if requestBytes, err = ioutil.ReadAll(req.Body); err != nil {
			r.JSON(400, data.APIError{Error: err.Error()})
			return
		}

		// Load schema from asset manager as bytes
		schemaBytes, err := Asset(schemaAssetName)
		if err != nil {
			r.JSON(400, data.APIError{Error: err.Error()})
			return
		}

		// Convert request to JSON
		err = json.Unmarshal(requestBytes, &requestObject)
		if err != nil {
			r.JSON(400, data.APIError{Error: err.Error()})
			return
		}

		// Convert schema to JSON
		err = json.Unmarshal(schemaBytes, &schemaObject)
		if err != nil {
			r.JSON(500, data.APIError{Error: err.Error()})
			return
		}

		// Initialize Schema based on JSON
		schema, err := gojsonschema.NewJsonSchemaDocument(schemaObject)
		if err != nil {
			r.JSON(500, data.APIError{Error: "Invalid Schema", Context: err.Error()})
			return
		}

		validationResult := schema.Validate(requestObject)

		if !validationResult.IsValid() {
			r.JSON(400, data.APIError{Error: "Validation Error", Context: validationResult.GetErrorMessages()})
			return
		}

		// Put this in the context so we can get it in the next handler.
		c.Map(JSONObject{
			Object: requestObject,
		})
	}
}

// Random DB functions
func genericIdInsert(db *mgo.Database, collection string, obj interface{}, r render.Render) {
	err := db.C(collection).Insert(obj)

	if err != nil && mgo.IsDup(err) {
		r.JSON(409, data.APIError{Error: "Already exists", Context: obj})
	}
}

func incrementCounter(db *mgo.Database, consumer, application, client, counter string, by int) error {
	var op string
	bucketName := fmt.Sprintf("counts.%s", counter)
	selector := bson.M{
		"consumer":    consumer,
		"application": application,
		"client":      client,
	}

	if by == 0 {
		op = "$set"
	} else {
		op = "$inc"
	}

	_, err := db.C("counters").Upsert(selector, bson.M{op: bson.M{bucketName: by}})
	return err
}

func getCounterForClient(db *mgo.Database, consumer, application, client string) (error, data.Counter) {
	var counter data.Counter
	// Get current client counts
	err := db.C("counters").Find(bson.M{
		"consumer":    consumer,
		"application": application,
		"client":      client,
	}).One(&counter)

	return err, counter
}

// Small utility functions
func restrictionQuerySelector(params martini.Params) bson.M {
	return bson.M{
		"consumer":    params["consumer"],
		"application": params["application"],
	}
}
