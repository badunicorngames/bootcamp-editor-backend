package appengine

import (
	"net/http"

	"bootcamp/editorservice/levels"
	"bootcamp/editorservice/territories"

	"github.com/gin-gonic/gin"
)

func Import() {
	// Tests need to reference this package, but don't actually need to do anything
}

func init() {
	// Initialize gin and set up middlewares
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	//router.Use(gin.Recovery())
	router.Use(allowOrigins())

	// Support OPTIONS for CORS
	router.OPTIONS("/*any", index)

	// Set up routes
	router.GET("/", index)
	levels.Init(router)
	territories.Init(router)

	// Tell AppEngine to forward all requests to gin
	http.Handle("/", router)
}

func index(context *gin.Context) {
	context.String(http.StatusOK, "hi\n")
}

// --- Allowed origins middleware

var allowedOrigins = map[string]bool{
	"localhost":                true,
	"test.badunicorngames.com": true,
}

func allowOrigins() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Next()
		return
	}
}
