package main

import (
	"log"
	"os"

	_ "github.com/L-ka080/DDB-6-case/docs"
	"github.com/L-ka080/DDB-6-case/internal/app"
)

// @title Exchange Chain API
// @version 1.0
// @description Backend API РїСЂРѕРµРєС‚Р° В«Р¦РµРїРѕС‡РєР° РѕР±РјРµРЅР°В».
//
// @contact.name Grigorev Vadim
// @contact.email grigorevvadim875@yandex.ru
//
// @host localhost:8080
// @BasePath /
// @schemes http
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Р’РІРµРґРёС‚Рµ С‚РѕРєРµРЅ РІ С„РѕСЂРјР°С‚Рµ: Bearer <access_token>
func main() {
	application, err := app.New()
	if err != nil {
		log.Printf("initialize application: %v", err)
		os.Exit(1)
	}

	if err := application.Run(); err != nil {
		log.Printf("application stopped with error: %v", err)
		os.Exit(1)
	}
}
