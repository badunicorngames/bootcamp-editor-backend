// package tests contains end-to-end tests
// this file tests the /levels route
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"appengine/aetest"

	main "bootcamp/editorservice/appengine"
)

// The test package must reference the main package.
// AppEngine does some magic so we don't need to actually do anything else with it.
var _ = main.Import

// --- Types and constants

type TestContext struct {
	t  *testing.T
	ae aetest.Instance
}

type Level struct {
	Key                 string             `json:"key,omitempty"`
	Parent              string             `json:"parent_key,omitempty"`
	Name                string             `json:"name,omitempty"`
	Rows                int32              `json:"rows,omitempty"`
	Columns             int32              `json:"columns,omitempty"`
	Duration            int32              `json:"duration,omitempty"`
	ComboTimer          float32            `json:"combo_timer,omitempty"`
	UnitDelayMultiplier float32            `json:"unit_delay_multiplier,omitempty"`
	MaxActiveUnits      int32              `json:"max_active_units,omitempty"`
	SpawnsPerSecond     float32            `json:"spawns_per_second,omitempty"`
	SpawnFrequency      map[string]float32 `json:"spawn_frequency,omitempty"`
}

const baseRoute = "/levels"

// Some test levels with all properties set
var testLevel1 = Level{
	Name:                "test level",
	Rows:                1,
	Columns:             2,
	Duration:            3,
	ComboTimer:          4.0,
	UnitDelayMultiplier: 5.0,
	MaxActiveUnits:      6,
	SpawnsPerSecond:     7.0,
	SpawnFrequency: map[string]float32{
		"grunt_fire": 1.0,
		"grunt_ice":  2.0,
	},
}

const testKey1 = "test_key_1"

var testLevel2 = Level{
	Name:                "test level 2",
	Rows:                2,
	Columns:             3,
	Duration:            4,
	ComboTimer:          5.0,
	UnitDelayMultiplier: 6.0,
	MaxActiveUnits:      7,
	SpawnsPerSecond:     8.0,
	SpawnFrequency: map[string]float32{
		"grunt_fire":     2.0,
		"grunt_ice_new":  3.0,
		"grunt_ice_new2": 4.0,
	},
}

const testKey2 = "test_key_2"

// --- Setup / Teardown

func setup(t *testing.T) *TestContext {
	t.Parallel()

	var options = aetest.Options{
		AppID: "testapp",
		StronglyConsistentDatastore: true,
	}
	ae, _ := aetest.NewInstance(&options)

	context := TestContext{
		t:  t,
		ae: ae,
	}

	return &context
}

func teardown(c *TestContext) {
	c.ae.Close()
}

// --- Tests

func TestGetWithMissingObjectFails(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Retrieve a level that hasn't been stored
	code, _ := loadLevelRaw(c, "nonExistingKey")
	assert.EqualValues(t, http.StatusNotFound, code)
}

func TestPutThenGetWithSameObjectMatches(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store and retrieve a level
	storeLevel(c, testKey1, testLevel1)
	level := loadLevel(c, testKey1)

	// Check that the key was applied
	assert.Equal(t, level.Key, testKey1)

	// Check that the objects match
	level.Key = ""
	assert.Equal(t, testLevel1, level)
}

func TestPutAndGetDifferentiateById(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store and retrieve two levels with different keys
	storeLevel(c, testKey1, testLevel1)
	storeLevel(c, testKey2, testLevel2)

	level1 := loadLevel(c, testKey1)
	level2 := loadLevel(c, testKey2)

	// Check that the objects match
	level1.Key = ""
	assert.Equal(t, testLevel1, level1)
	level2.Key = ""
	assert.Equal(t, testLevel2, level2)
}

func TestPutUpdatesExistingEntity(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store and overwrite a level
	storeLevel(c, testKey1, testLevel1)
	storeLevel(c, testKey1, testLevel2)

	level := loadLevel(c, testKey1)

	// Check that the returned object matches the newer object
	level.Key = ""
	assert.Equal(t, testLevel2, level)
}

func TestDeleteWithMissingObjectSucceeds(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Delete a level that hasn't been stored
	// It doesn't 404, and that's fine. It shouldn't matter.
	// Datastore is returning success behind the scenes, and changing that
	// would require doing get+delete which right now is needlessly expensive.
	deleteLevel(c, "nonExistingKey")
	// asserts in the helper
}

func TestDeleteWithExistingObjectSucceeds(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store the level and read it back (should 200)
	storeLevel(c, testKey1, testLevel1)
	_ = loadLevel(c, testKey1)

	// Delete the level and read it back again (should 404)
	deleteLevel(c, testKey1)
	code, _ := loadLevelRaw(c, testKey1)
	assert.EqualValues(t, http.StatusNotFound, code)
}

func TestDeleteDifferentiatesById(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store two levels with different keys
	storeLevel(c, testKey1, testLevel1)
	storeLevel(c, testKey2, testLevel2)

	// Delete one
	deleteLevel(c, testKey2)

	// Make sure the first level still loads
	_ = loadLevel(c, testKey1)

	// Make sure the deleted one 404s
	code, _ := loadLevelRaw(c, testKey2)
	assert.EqualValues(t, http.StatusNotFound, code)

	// Make sure the deleted one doesn't show up in a query
	levels := queryAll(c)
	assert.EqualValues(t, 1, len(levels))
	assert.Equal(t, testKey1, levels[0].Key)
}

func TestQueryWithNoLevelsSucceeds(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	levels := queryAll(c)
	// asserts in the helper

	// Should have zero results
	assert.EqualValues(t, 0, len(levels))
}

func TestQueryRetrievesAllLevels(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store two levels
	storeLevel(c, testKey1, testLevel1)
	storeLevel(c, testKey2, testLevel2)

	levels := queryAll(c)

	// Put the results into a map so they're easier to work with
	// This also de-dupes if the service re-uses a key
	levelsMap := make(map[string]Level)
	for _, level := range levels {
		levelsMap[level.Key] = level
	}

	// Result should have two items
	assert.EqualValues(t, 2, len(levelsMap))

	// And they should match the originals
	level1 := levelsMap[testKey1]
	level1.Key = ""
	assert.Equal(t, testLevel1, level1)

	level2 := levelsMap[testKey2]
	level2.Key = ""
	assert.Equal(t, testLevel2, level2)
}

func TestQueryLimitsResults(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store 101 levels
	for i := 0; i < 101; i++ {
		testKey := fmt.Sprintf("test_key_%d", i)
		storeLevel(c, testKey, testLevel1)
	}

	// Query all should only return 100
	levels := queryAll(c)
	assert.EqualValues(t, 100, len(levels))
}

func TestGetWithValidParentInheritsProperties(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	parentKey := testKey1
	parentLevel := testLevel1
	childKey := testKey2

	// Store a level. This will become the parent.
	storeLevel(c, parentKey, parentLevel)

	// Store an otherwise-empty level with just its parent and name set
	childName := "child name"
	childLevel := Level{Parent: parentKey, Name: childName}
	storeLevel(c, childKey, childLevel)

	// Retrieve the child level.
	level := loadLevel(c, childKey)

	// Key should have been set by the back-end.
	assert.Equal(t, testKey2, level.Key)
	level.Key = ""
	assert.Equal(t, parentKey, level.Parent)
	level.Parent = ""

	// Name should be what we set it to (not overwritten by the parent)
	assert.Equal(t, childName, level.Name)
	level.Name = parentLevel.Name

	// All other properties should be equal to the parent's properties
	assert.Equal(t, parentLevel, level)
}

func TestQueryWithValidParentInheritsProperties(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	parentKey := testKey1
	parentLevel := testLevel1
	childKey := testKey2

	// Store a level. This will become the parent.
	storeLevel(c, parentKey, parentLevel)

	// Store an otherwise-empty level with just its parent and name set
	childName := "child name"
	childLevel := Level{Parent: parentKey, Name: childName}
	storeLevel(c, childKey, childLevel)

	// Retrieve the child level.
	levels := queryAll(c)
	var level Level
	foundLevel := false
	for _, element := range levels {
		if element.Key == childKey {
			level = element
			foundLevel = true
			break
		}
	}
	assert.EqualValues(t, true, foundLevel)

	// Key should have been set by the back-end.
	assert.Equal(t, testKey2, level.Key)
	level.Key = ""
	assert.Equal(t, parentKey, level.Parent)
	level.Parent = ""

	// Name should be what we set it to (not overwritten by the parent)
	assert.Equal(t, childName, level.Name)
	level.Name = parentLevel.Name

	// All other properties should be equal to the parent's properties
	assert.Equal(t, parentLevel, level)
}

func TestGetWithMissingParentFails(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store a level with its parent set
	parentKey := "invalid_key"
	childKey := testKey1
	childLevel := testLevel1
	childLevel.Parent = parentKey
	storeLevel(c, childKey, childLevel)

	// Retrieve the child level. It should error.
	code, _ := loadLevelRaw(c, childKey)
	assert.EqualValues(t, http.StatusNotFound, code)
}

func TestUpdateParentAlsoUpdatesChildren(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store a parent
	parentKey := testKey1
	parentLevel := testLevel1
	storeLevel(c, parentKey, parentLevel)

	// Store a child referencing the parent
	childKey := testKey2
	childLevel := Level{Parent: parentKey}
	storeLevel(c, childKey, childLevel)

	// Get the child. This is important because it will trigger caching of the child.
	level := loadLevel(c, childKey)
	level.Parent = ""
	level.Key = ""
	assert.Equal(t, parentLevel, level)

	// Update the parent
	parentLevel.Name = "Updated Name"
	storeLevel(c, parentKey, parentLevel)

	// Get the child again. It should have the updated parent properties.
	level = loadLevel(c, childKey)
	level.Parent = ""
	level.Key = ""
	assert.Equal(t, parentLevel, level)
}

// --- Helpers

func buildQueryRoute() string {
	return baseRoute
}

func buildEntityRoute(id string) string {
	return baseRoute + "/" + id
}

func invoke(c *TestContext, verb string, path string, obj interface{}) (code int, response string) {
	marshalledObj, _ := json.Marshal(obj)
	request, _ := c.ae.NewRequest(verb, path, bytes.NewBuffer(marshalledObj))
	w := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(w, request)
	body, _ := ioutil.ReadAll(w.Body)

	code = w.Code
	response = string(body)

	c.t.Logf("%s %s\ncode: %+v\nresponse: %+v\n", verb, path, code, response)
	return
}

func storeLevel(c *TestContext, id string, level Level) (int, string) {
	code, response := invoke(c, "PUT", buildEntityRoute(id), level)
	assert.EqualValues(c.t, http.StatusOK, code)

	// Populate the caches (help catch cache invalidation errors)
	loadLevelRaw(c, id)
	queryAll(c)

	return code, response
}

func loadLevel(c *TestContext, id string) (level Level) {
	code, resp := invoke(c, "GET", buildEntityRoute(id), nil)
	assert.EqualValues(c.t, http.StatusOK, code)

	json.Unmarshal([]byte(resp), &level)
	return
}

func loadLevelRaw(c *TestContext, id string) (int, string) {
	code, response := invoke(c, "GET", buildEntityRoute(id), nil)
	return code, response
}

func deleteLevel(c *TestContext, id string) (int, string) {
	code, response := invoke(c, "DELETE", buildEntityRoute(id), nil)
	assert.EqualValues(c.t, http.StatusOK, code)
	return code, response
}

func queryAll(c *TestContext) (levels []Level) {
	code, resp := invoke(c, "GET", buildQueryRoute(), nil)
	assert.EqualValues(c.t, http.StatusOK, code)

	json.Unmarshal([]byte(resp), &levels)
	return
}
