// Package levels serves the levels resource.
package levels

import (
	"encoding/json"
	"net/http"

	"appengine"
	"appengine/datastore"

	"github.com/gin-gonic/gin"

	"bootcamp/editorservice/cache"
	"bootcamp/editorservice/levels/level"
)

// --- Types and constants

const kind string = "Level"
const queryAllKey string = "query:all@levels"

// All levels share a single entity root.  This is important because this provides
// strong consistency for levels.
const levelRootKeyName string = "LevelRoot"

var levelRootKey *datastore.Key

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

// --- Level cache

type levelCacheEntry level.DatastoreLevel

func (entry *levelCacheEntry) GetCacheKey() string {
	return "level:" + entry.Key
}

func (entry *levelCacheEntry) MarshalBinary() ([]byte, error) {
	return json.Marshal(entry)
}

func (entry *levelCacheEntry) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, entry)
}

// --- Route handlers

// Init sets up routes for this resource
func Init(router *gin.Engine) {
	router.GET("/levels/:id", handleGet)
	router.POST("/levels/:id", handlePost)
	router.PUT("/levels/:id", handlePut)
	router.DELETE("/levels/:id", handleDelete)
	router.GET("/levels", handleQuery)
}

func handleGet(context *gin.Context) {
	path := context.Request.URL.Path
	levelId := context.Param("id")
	appengineContext := appengine.NewContext(context.Request)

	// Check response cache
	cachedResponse := &responseCacheEntry{Path: path}
	err := cache.GetCachedResource(appengineContext, cachedResponse)
	if err == nil {
		context.JSON(cachedResponse.Code, cachedResponse.Response)
		return
	}

	// Fetch from level cache or datastore
	result, err := getLevel(levelId, appengineContext)
	if err == datastore.ErrNoSuchEntity {
		cacheEntry := responseCacheEntry{
			Path:     path,
			Code:     http.StatusNotFound,
			Response: "Level does not exist",
		}
		cache.CacheResource(appengineContext, &cacheEntry)
		context.String(cacheEntry.Code, cacheEntry.Response.(string))
		return
	} else if err != nil {
		context.String(http.StatusInternalServerError, "Could not retrieve the level: %+v\n", err)
		return
	}

	// If we got this far, then we found the level
	// Cache and return the result
	cacheEntry := &responseCacheEntry{
		Path:     path,
		Code:     http.StatusOK,
		Response: (*level.DatastoreLevel)(result).ToJsonLevel(),
	}
	cache.CacheResource(appengineContext, cacheEntry)

	context.JSON(cacheEntry.Code, cacheEntry.Response)
}

func handlePost(context *gin.Context) {
	var level level.JsonLevel

	// Unmarshal to JsonLevel
	err := context.BindJSON(&level)
	if err != nil {
		context.String(http.StatusBadRequest, "Failed to unmarshal the JSON: %+v\n", err)
		return
	}

	// The level key/id must come from the URL path
	level.Key = new(string)
	*level.Key = context.Param("id")

	// Write to datastore
	dsLevel := level.ToDatastoreLevel()
	appengineContext := appengine.NewContext(context.Request)
	_, err = datastore.Put(appengineContext, makeDatastoreKey(appengineContext, dsLevel.Key), dsLevel)
	if err != nil {
		context.String(http.StatusInternalServerError, "Failed to store the level: %+v", err)
		return
	}

	// Invalidate everything
	invalidateLevelCaches(appengineContext, dsLevel.Key)
	invalidateChildLevelCaches(appengineContext, dsLevel.Key)
	invalidateQueryCaches(appengineContext)

	context.JSON(http.StatusOK, nil)
}

func handlePut(context *gin.Context) {
	handlePost(context)
}

func handleDelete(context *gin.Context) {
	levelId := context.Param("id")
	appengineContext := appengine.NewContext(context.Request)

	// Delete from datastore
	err := datastore.Delete(appengineContext, makeDatastoreKey(appengineContext, levelId))
	if err != nil {
		context.String(http.StatusInternalServerError, "Failed to delete the level: %+v", err)
		return
	}

	// Invalidate everything
	invalidateLevelCaches(appengineContext, levelId)
	invalidateChildLevelCaches(appengineContext, levelId)
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

	// Query to get a list of level keys
	query := datastore.NewQuery(kind).Ancestor(getLevelRootKey(appengineContext)).Limit(100).KeysOnly()
	keys, err := query.GetAll(appengineContext, nil)

	// Load each level by its key
	// We have to do it this way in order to resolve the parent-child relationships.
	var dsResults []level.DatastoreLevel
	for _, element := range keys {
		resolvedLevel, err := getLevel(element.StringID(), appengineContext)
		if err == nil {
			dsResults = append(dsResults, (level.DatastoreLevel)(*resolvedLevel))
		}
	}

	var response []*level.JsonLevel
	for _, element := range dsResults {
		response = append(response, (&element).ToJsonLevel())
	}

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

func getLevel(levelId string, appengineContext appengine.Context) (*levelCacheEntry, error) {

	// Check level cache
	result := &levelCacheEntry{Key: levelId}
	err := cache.GetCachedResource(appengineContext, result)

	// Check datastore if necessary
	if err != nil {
		result = &levelCacheEntry{}
		err = datastore.Get(appengineContext, makeDatastoreKey(appengineContext, levelId), result)
		if err != nil {
			return nil, err
		} else /*err == nil. level was found.*/ {
			// Level loaded from datastore will not yet have its parent's properties applied,
			// so we need to fetch the parent and do that.
			if result.HasParent && len(result.Parent) > 0 {
				parentLevel, err := getLevel(result.Parent, appengineContext)
				if err != nil {
					return nil, err
				}

				(*level.DatastoreLevel)(result).MergeParentProperties((*level.DatastoreLevel)(parentLevel))
			}

			// Cache the finalized level object with its parent's properties applied
			cache.CacheResource(appengineContext, result)
		}
	}

	return result, nil
}

func buildResourcePath(levelId string) string {
	return "/levels/" + levelId
}

func invalidateChildLevelCaches(context appengine.Context, parentId string) {
	// Query to find children
	query := datastore.NewQuery(kind).Ancestor(getLevelRootKey(context)).Filter("Parent =", parentId).KeysOnly()
	keys, err := query.GetAll(context, nil)
	if err != nil {
		return
	}

	// Invalidate each one
	for _, element := range keys {
		invalidateLevelCaches(context, element.StringID())
	}
}

func invalidateLevelCaches(context appengine.Context, levelId string) {
	// Response cache
	responseEntry := &responseCacheEntry{Path: buildResourcePath(levelId)}
	cache.InvalidateCacheEntry(context, responseEntry)

	// Level cache
	levelEntry := &levelCacheEntry{Key: levelId}
	cache.InvalidateCacheEntry(context, levelEntry)
}

func invalidateQueryCaches(context appengine.Context) {
	// Query-all cache
	queryAllEntry := &responseCacheEntry{Path: queryAllKey}
	cache.InvalidateCacheEntry(context, queryAllEntry)
}

func getLevelRootKey(context appengine.Context) *datastore.Key {
	if levelRootKey != nil {
		return levelRootKey
	}

	levelRootKey = datastore.NewKey(context, kind, levelRootKeyName, 0, nil)
	return levelRootKey
}

func makeDatastoreKey(context appengine.Context, key string) *datastore.Key {
	return datastore.NewKey(context, kind, key, 0, getLevelRootKey(context))
}
