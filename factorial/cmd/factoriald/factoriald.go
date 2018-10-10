package main

import (
	"fmt"
	"math/big"
	"net/http"
	"log"
	"strconv"
)



func factorial(fac int64) string {
	x := new(big.Int)
	x.MulRange(1, fac)
	return fmt.Sprint(x)
}


func handler(w http.ResponseWriter, r *http.Request){
	fmt.Println("Got", r.URL.Path)

	numString := r.URL.Query().Get("fac")
	num, err := strconv.ParseInt(numString, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad reqest"))
		return
	}


	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", factorial(num))
}


func main(){
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
