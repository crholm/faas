package main

import (
	"net/http"
	"log"
	"strings"
	"fmt"
	"github.com/crholm/faas/faas-gateway/cmd/gatewayd/routes"
)


var routing *routes.Routing


func writeError(w http.ResponseWriter, code int, message string){
	w.WriteHeader(code)
	fmt.Fprintf(w, message)
}


func proxy(w http.ResponseWriter, r *http.Request){

	if !strings.HasPrefix(r.URL.Path, "/faas/"){
		writeError(w, http.StatusBadRequest, "must have prefix faas to call function")
		return
	}

	functionName := strings.TrimPrefix(r.URL.Path, "/faas/")
	routing.Proxy(functionName, w, r)



}


func main(){

	routing = routes.New("faas.")
	routing.Start()


	http.HandleFunc("/", proxy)
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		routing.Stop()
		log.Fatal("ListenAndServe: ", err)
	}
}
