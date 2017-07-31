package main

import (
	"fmt"
	"image/png"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"

	"github.com/scristofari/image-server/part6/resizer"
)

func main() {
	r := handlers()

	port := os.Getenv("PORT")
	if port == "" {
		panic(fmt.Sprint("failed to get PORT env, defined ?"))
	}

	log.Printf("Listening on port %s ...", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func handlers() http.Handler {
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/access/token", authBasicHandleFunc(accessHandleFunc)).Methods("GET")
	r.HandleFunc("/upload/{jwt}", jwtHandleFunc(uploadHandleFunc)).Methods("POST")
	r.HandleFunc("/images/{img}", imageHandleFunc).Methods("GET")
	return r
}

func accessHandleFunc(w http.ResponseWriter, r *http.Request) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		ExpiresAt: time.Now().Add(time.Minute * 1).Unix(),
	})
	t, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate the token: %s", err.Error()), http.StatusBadRequest)
		return
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	w.Write([]byte(fmt.Sprintf("%s://%s/upload/%s", scheme, r.Host, t)))
}

func uploadHandleFunc(w http.ResponseWriter, r *http.Request) {
	// Prevent from too large uploaded file / PART 4
	r.Body = http.MaxBytesReader(w, r.Body, int64(resizer.UploadMaxSize))

	ctx := r.Context()
	timeout := time.After(1 * time.Minute)
	resultChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		image, _, err := r.FormFile("image")
		if err != nil {
			errorChan <- err
			return
		}
		defer image.Close()

		uuid, err := resizer.Uploadfile(&resizer.DiskProvider{}, image)
		if err != nil {
			errorChan <- err
			return
		}

		resultChan <- uuid
	}()

	defer close(resultChan)
	defer close(errorChan)
	for {
		select {
		case <-ctx.Done():
			http.Error(w, "Cancel", http.StatusBadRequest)
			return
		case <-timeout:
			http.Error(w, "Timeout", http.StatusBadRequest)
			return
		case uuid := <-resultChan:
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(fmt.Sprintf("%s://%s/images/%s.png", scheme, r.Host, uuid)))
			return
		case err := <-errorChan:
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
}

func imageHandleFunc(w http.ResponseWriter, r *http.Request) {
	/** vars from gorilla mux empty, in test case, we do not execute the router */
	hash := strings.Split(r.URL.Path, "/")

	q, err := resizer.GetQueryFromURL(r.URL)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get the query: %s", err.Error()), http.StatusBadRequest)
		return
	}

	i, err := resizer.Resize(&resizer.DiskProvider{}, hash[2], q)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resize the image: %s", err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Cache-Control", "max-age=3600")
	png.Encode(w, i)
}

func authBasicHandleFunc(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "failed to get auth basic credentials", http.StatusForbidden)
			return
		}

		err := resizer.CheckCredentials(user, pass)
		if err != nil {
			http.Error(w, "failed to sign in: "+err.Error(), http.StatusForbidden)
			return
		}

		f(w, r)
	}
}

func jwtHandleFunc(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		/** vars from gorilla mux empty, in test case, we do not execute the router */
		hash := strings.Split(r.URL.Path, "/")

		token, err := jwt.Parse(hash[2], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(os.Getenv("JWT_SECRET")), nil
		})
		if err != nil {
			http.Error(w, "failed to authenticate: "+err.Error(), http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			http.Error(w, "failed to authenticate, token not valid", http.StatusUnauthorized)
			return
		}

		f(w, r)
	}
}
