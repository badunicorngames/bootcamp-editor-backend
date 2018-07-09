// package tests contains end-to-end tests
// this file tests the /territories route
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

type Territory struct {
	Id       string   `json:"id,omitempty"`
	Sequence int32    `json:"sequence,omitempty"`
	Name     string   `json:"name,omitempty"`
	Levels   []string `json:"levels"`
}

const baseRoute = "/territories"

// Some test territories with all properties set
var testTerritory1 = Territory{
	Sequence: 1,
	Name:     "test territory",
	Levels: []string{
		"test",
		"default",
	},
}

const testKey1 = "test_key_1"

var testTerritory2 = Territory{
	Sequence: 2,
	Name:     "test territory 2",
	Levels: []string{
		"default",
		"other_one",
		"something_else",
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

	// Retrieve a territory that hasn't been stored
	code, _ := loadTerritoryRaw(c, "nonExistingKey")
	assert.EqualValues(t, http.StatusNotFound, code)
}

func TestPutThenGetWithSameObjectMatches(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store and retrieve a territory
	storeTerritory(c, testKey1, testTerritory1)
	territory := loadTerritory(c, testKey1)

	// Check that the key was applied
	assert.Equal(t, territory.Id, testKey1)

	// Check that the objects match
	territory.Id = ""
	assert.Equal(t, testTerritory1, territory)
}

func TestPutAndGetDifferentiateById(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store and retrieve two territories with different keys
	storeTerritory(c, testKey1, testTerritory1)
	storeTerritory(c, testKey2, testTerritory2)

	territory1 := loadTerritory(c, testKey1)
	territory2 := loadTerritory(c, testKey2)

	// Check that the objects match
	territory1.Id = ""
	assert.Equal(t, testTerritory1, territory1)
	territory2.Id = ""
	assert.Equal(t, testTerritory2, territory2)
}

func TestPutUpdatesExistingEntity(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store and overwrite a territory
	storeTerritory(c, testKey1, testTerritory1)
	storeTerritory(c, testKey1, testTerritory2)

	territory := loadTerritory(c, testKey1)

	// Check that the returned object matches the newer object
	territory.Id = ""
	assert.Equal(t, testTerritory2, territory)
}

func TestDeleteWithMissingObjectSucceeds(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Delete a territory that hasn't been stored
	// It doesn't 404, and that's fine. It shouldn't matter.
	// Datastore is returning success behind the scenes, and changing that
	// would require doing get+delete which right now is needlessly expensive.
	deleteTerritory(c, "nonExistingKey")
	// asserts in the helper
}

func TestDeleteWithExistingObjectSucceeds(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store the territory and read it back (should 200)
	storeTerritory(c, testKey1, testTerritory1)
	_ = loadTerritory(c, testKey1)

	// Delete the territory and read it back again (should 404)
	deleteTerritory(c, testKey1)
	code, _ := loadTerritoryRaw(c, testKey1)
	assert.EqualValues(t, http.StatusNotFound, code)
}

func TestDeleteDifferentiatesById(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store two territories with different keys
	storeTerritory(c, testKey1, testTerritory1)
	storeTerritory(c, testKey2, testTerritory2)

	// Delete one
	deleteTerritory(c, testKey2)

	// Make sure the first territory still loads
	_ = loadTerritory(c, testKey1)

	// Make sure the deleted one 404s
	code, _ := loadTerritoryRaw(c, testKey2)
	assert.EqualValues(t, http.StatusNotFound, code)

	// Make sure the deleted one doesn't show up in a query
	territories := queryAll(c)
	assert.EqualValues(t, 1, len(territories))
	assert.Equal(t, testKey1, territories[0].Id)
}

func TestQueryWithNoTerritoriesSucceeds(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	territories := queryAll(c)
	// asserts in the helper

	// Should have zero results
	assert.EqualValues(t, 0, len(territories))
}

func TestQueryRetrievesAllTerritories(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store two territories
	storeTerritory(c, testKey1, testTerritory1)
	storeTerritory(c, testKey2, testTerritory2)

	territories := queryAll(c)

	// Put the results into a map so they're easier to work with
	// This also de-dupes if the service re-uses a key
	territoriesMap := make(map[string]Territory)
	for _, territory := range territories {
		territoriesMap[territory.Id] = territory
	}

	// Result should have two items
	assert.EqualValues(t, 2, len(territoriesMap))

	// And they should match the originals
	territory1 := territoriesMap[testKey1]
	territory1.Id = ""
	assert.Equal(t, testTerritory1, territory1)

	territory2 := territoriesMap[testKey2]
	territory2.Id = ""
	assert.Equal(t, testTerritory2, territory2)
}

func TestQueryLimitsResults(t *testing.T) {
	c := setup(t)
	defer teardown(c)

	// Store 101 territories
	for i := 0; i < 101; i++ {
		testKey := fmt.Sprintf("test_key_%d", i)
		storeTerritory(c, testKey, testTerritory1)
	}

	// Query all should only return 100
	territories := queryAll(c)
	assert.EqualValues(t, 100, len(territories))
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

func storeTerritory(c *TestContext, id string, territory Territory) (int, string) {
	code, response := invoke(c, "PUT", buildEntityRoute(id), territory)
	assert.EqualValues(c.t, http.StatusOK, code)

	// Populate the caches (help catch cache invalidation errors)
	loadTerritoryRaw(c, id)
	queryAll(c)

	return code, response
}

func loadTerritory(c *TestContext, id string) (territory Territory) {
	code, resp := invoke(c, "GET", buildEntityRoute(id), nil)
	assert.EqualValues(c.t, http.StatusOK, code)

	json.Unmarshal([]byte(resp), &territory)
	return
}

func loadTerritoryRaw(c *TestContext, id string) (int, string) {
	code, response := invoke(c, "GET", buildEntityRoute(id), nil)
	return code, response
}

func deleteTerritory(c *TestContext, id string) (int, string) {
	code, response := invoke(c, "DELETE", buildEntityRoute(id), nil)
	assert.EqualValues(c.t, http.StatusOK, code)
	return code, response
}

func queryAll(c *TestContext) (territories []Territory) {
	code, resp := invoke(c, "GET", buildQueryRoute(), nil)
	assert.EqualValues(c.t, http.StatusOK, code)

	json.Unmarshal([]byte(resp), &territories)
	return
}
