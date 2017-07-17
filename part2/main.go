package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

var (
	outputDir = "images"
)

func main() {
	http.Handle("/", handlers())

	log.Printf("Listening on port 8080 ...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handlers() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/upload", uploadHandler).Methods("POST")
	r.HandleFunc("/images/{img}", imageHandler).Methods("GET")

	return r
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	image, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get the image: %s", err.Error()), http.StatusBadRequest)
		return
	}
	defer image.Close()

	f, err := os.OpenFile(outputDir+"/"+header.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	io.Copy(f, image)
	w.WriteHeader(http.StatusCreated)
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	image, err := ioutil.ReadFile(outputDir + "/" + vars["img"])
	if err != nil {
		http.Error(w, "image not found : "+err.Error(), http.StatusNotFound)
		return
	}
	w.Write(image)
}
