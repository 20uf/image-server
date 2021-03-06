package main

import (
	"bytes"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestUploadImage(t *testing.T) {
	body, contentType, err := loadFormFile("golang.png")
	if err != nil {
		t.Error(err.Error())
		return
	}

	r, _ := http.NewRequest("POST", "http://localhost/images/golang.png", body)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	uploadHandler(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected %d, get %d", http.StatusCreated, w.Code)
		t.Error(w.Body)
	}
}

func TestGetImage(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost/images/golang.png", nil)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	w := httptest.NewRecorder()

	imageHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected %d, get %d", http.StatusOK, w.Code)
		t.Error(w.Body)
	}
}

func loadFormFile(path string) (*bytes.Buffer, string, error) {
	file, err := os.Open("golang.png")
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, "", err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, "", err
	}

	body := new(bytes.Buffer)
	bw := multipart.NewWriter(body)

	fw, err := bw.CreateFormFile("image", fi.Name())
	if err != nil {
		return nil, "", err
	}
	fw.Write(fileContents)

	err = bw.Close()
	if err != nil {
		return nil, "", err
	}

	return body, bw.FormDataContentType(), nil
}
