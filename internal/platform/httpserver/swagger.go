package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files/v2"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/net/webdav"
)

func SwaggerHandler() gin.HandlerFunc {
	dynamicHandler := ginSwagger.WrapHandler(&webdav.Handler{
		FileSystem: webdav.NewMemFS(),
		LockSystem: webdav.NewMemLS(),
	})
	staticHandler := http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFiles.FS)))

	return func(c *gin.Context) {
		switch c.Param("any") {
		case "", "/":
			c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
		case "/index.html", "/index.css", "/swagger-initializer.js", "/doc.json":
			dynamicHandler(c)
		default:
			staticHandler.ServeHTTP(c.Writer, c.Request)
		}
	}
}
