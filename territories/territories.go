// Package territories serves the territories resource.
package territories

import (
	"encoding/json"
	"net/http"

	"appengine"
	"appengine/datastore"

	"github.com/gin-gonic/gin"

	"bootcamp/editorservice/cache"
	"bootcamp/editorservice/territories/territory"
)

// --- Types and constants

const kind string = "Territory"
const queryAllKey string = "query:all@territories"

// All territories share a single entity root.  This isn't really important.
const territoryRootKeyName string = "TerritoryRoot"

var territoryRootKey *datastore.Key

// -- Response cache

type responseCacheEntry struct {
	Path     string
	Code     int
	Response interface{}
}

func (entry *responseCacheEntry) GetCacheKey() string {
	return "response:" + entry.Path
}

func (entry *responseCacheEntry) MarshalBinary() ([]byte, error) {
	return json.Marshal(entry)
}

func (entry *responseCacheEntry) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, entry)
}

// --- Route handlers

// Init sets up routes for this resource
func Init(router *gin.Engine) {
	router.GET("/territories/:id", handleGet)
	router.POST("/territories/:id", handlePost)
	router.PUT("/territories/:id", handlePut)
	router.DELETE("/territories/:id", handleDelete)
	router.GET("/territories", handleQuery)
}

func handleGet(context *gin.Context) {
	path := context.Request.URL.Path
	territoryId := context.Param("id")
	appengineContext := appengine.NewContext(context.Request)
	result := &territory.Territory{}

	// Check response cache
	cachedResponse := &responseCacheEntry{Path: path}
	err := cache.GetCachedResource(appengineContext, cachedResponse)
	if err == nil {
		context.JSON(cachedResponse.Code, cachedResponse.Response)
		return
	} else /*err != nil*/ {
		// Check datastore
		err = datastore.Get(appengineContext, makeDatastoreKey(appengineContext, territoryId), result)
		if err == datastore.ErrNoSuchEntity {
			cacheEntry := responseCacheEntry{
				Path:     path,
				Code:     http.StatusNotFound,
				Response: "Territory does not exist",
			}
			cache.CacheResource(appengineContext, &cacheEntry)
			context.String(cacheEntry.Code, cacheEntry.Response.(string))
			return
		} else if err != nil {
			context.String(http.StatusInternalServerError, "Could not retrieve the territory: %+v\n", err)
			return
		}
	}

	// If we got this far, then we found the territory
	// Cache and return the result
	cacheEntry := &responseCacheEntry{
		Path:     path,
		Code:     http.StatusOK,
		Response: result,
	}
	cache.CacheResource(appengineContext, cacheEntry)

	context.JSON(cacheEntry.Code, cacheEntry.Response)
}

func handlePost(context *gin.Context) {
	var territory territory.Territory

	// Unmarshal
	err := context.BindJSON(&territory)
	if err != nil {
		context.String(http.StatusBadRequest, "Failed to unmarshal the JSON: %+v\n", err)
		return
	}

	// The territory id must come from the URL path
	territory.Id = new(string)
	*territory.Id = context.Param("id")

	// Write to datastore
	appengineContext := appengine.NewContext(context.Request)
	_, err = datastore.Put(appengineContext, makeDatastoreKey(appengineContext, *territory.Id), &territory)
	if err != nil {
		context.String(http.StatusInternalServerError, "Failed to store the territory: %+v", err)
		return
	}

	// Invalidate everything
	invalidateResponseCache(appengineContext, *territory.Id)
	invalidateQueryCaches(appengineContext)

	context.JSON(http.StatusOK, nil)
}

func handlePut(context *gin.Context) {
	handlePost(context)
}

func handleDelete(context *gin.Context) {
	territoryId := context.Param("id")
	appengineContext := appengine.NewContext(context.Request)

	// Delete from datastore
	err := datastore.Delete(appengineContext, makeDatastoreKey(appengineContext, territoryId))
	if err != nil {
		context.String(http.StatusInternalServerError, "Failed to delete the territory: %+v", err)
		return
	}

	// Invalidate everything
	invalidateResponseCache(appengineContext, territoryId)
	invalidateQueryCaches(appengineContext)

	context.JSON(http.StatusOK, nil)
}

func handleQuery(context *gin.Context) {
	appengineContext := appengine.NewContext(context.Request)

	// Check response cache
	responseEntry := &responseCacheEntry{Path: queryAllKey}
	err := cache.GetCachedResource(appengineContext, responseEntry)
	if err == nil {
		context.JSON(responseEntry.Code, responseEntry.Response)
		return
	}

	// Query to get all the territories
	var response []*territory.Territory
	query := datastore.NewQuery(kind).Ancestor(getTerritoryRootKey(appengineContext)).Limit(100)
	_, err = query.GetAll(appengineContext, &response)

	// Cache and return the result
	cacheEntry := &responseCacheEntry{
		Path:     queryAllKey,
		Code:     http.StatusOK,
		Response: response,
	}
	cache.CacheResource(appengineContext, cacheEntry)
	context.JSON(cacheEntry.Code, cacheEntry.Response)
}

// --- Helpers

func buildResourcePath(territoryId string) string {
	return "/territories/" + territoryId
}

func invalidateResponseCache(context appengine.Context, territoryId string) {
	responseEntry := &responseCacheEntry{Path: buildResourcePath(territoryId)}
	cache.InvalidateCacheEntry(context, responseEntry)
}

func invalidateQueryCaches(context appengine.Context) {
	// Query-all cache
	queryAllEntry := &responseCacheEntry{Path: queryAllKey}
	cache.InvalidateCacheEntry(context, queryAllEntry)
}

func getTerritoryRootKey(context appengine.Context) *datastore.Key {
	if territoryRootKey != nil {
		return territoryRootKey
	}

	territoryRootKey = datastore.NewKey(context, kind, territoryRootKeyName, 0, nil)
	return territoryRootKey
}

func makeDatastoreKey(context appengine.Context, key string) *datastore.Key {
	return datastore.NewKey(context, kind, key, 0, getTerritoryRootKey(context))
}
