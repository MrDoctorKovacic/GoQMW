package system

import "github.com/gorilla/mux"

// Mod implements MDroid module and exports
var Mod *stat

func (*stat) Setup(configAddr *map[string]string) {

}

func (*stat) SetRoutes(router *mux.Router) {
	router.HandleFunc("/system/", HandleGetAll).Methods("GET")
	router.HandleFunc("/system/{name}", HandleGet).Methods("GET")
	router.HandleFunc("/system/{name}", HandleSet).Methods("POST")
}
