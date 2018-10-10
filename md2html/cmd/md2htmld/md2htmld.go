package main

import (
	"fmt"
	"net/http"
	"log"
	"io/ioutil"
	"gopkg.in/russross/blackfriday.v2"
)




func handler(w http.ResponseWriter, r *http.Request){
	fmt.Println("Got", r.URL.Path)
	md, err := ioutil.ReadAll(r.Body)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad reqest, no body"))
		return
	}

	fmt.Fprintf(w, "%s", blackfriday.Run(md))
}

func main(){
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
