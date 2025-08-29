package docs

import (
	"embed"
	"log"
	"net/http"

	"github.com/swaggo/swag"
)

//go:embed swagger.json
var swaggerFS embed.FS

var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "/",
	Schemes:          []string{"http"},
	Title:            "RPC Forwarder API",
	Description:      "RPC API Definition",
	InfoInstanceName: "swagger",
}

func init() {
	data, err := swaggerFS.ReadFile("swagger.json")
	if err != nil {
		log.Fatalf("failed to load swagger.json: %v", err)
	}
	SwaggerInfo.SwaggerTemplate = string(data)
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}

func JSONHandler(w http.ResponseWriter, r *http.Request) {
	data, err := swaggerFS.ReadFile("swagger.json")
	if err != nil {
		http.Error(w, "swagger spec not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
