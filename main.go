package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	MAX_FILE_SIZE = 2 * 1024 * 1024 // 2MB
	IMAGE_DIR     = "uploads/images"
)

func uploadImage(w http.ResponseWriter, r *http.Request) {
	// Maximum upload of 10 MB files in memory buffer
	r.ParseMultipartForm(10 * 1024 * 1024)

	// Get handler for filename, size and headers
	file, fileInfo, err := r.FormFile("image")
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// validation criterias
	// validate file type
	if !strings.Contains(fileInfo.Header.Get("Content-Type"), "image/") {
		w.Write([]byte("Only image file is allowed"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//validate image size
	if fileInfo.Size > MAX_FILE_SIZE {
		w.Write([]byte("Max image size is 2MB"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer file.Close()

	// folder path pattern : year/month/day
	// file location pattern : file_name-folder_path.ext

	now := time.Now().UTC()
	unixTime := now.UnixNano()
	dateStr := fmt.Sprintf("%d%02d%02d", now.Year(), now.Month(), now.Day())
	fileExtension := path.Ext(fileInfo.Filename)
	newFileName := fmt.Sprintf("%d%s", unixTime, fileExtension)                        // add a random
	folderHierarchy := fmt.Sprintf("%d/%02d/%02d", now.Year(), now.Month(), now.Day()) // splitted dateStr
	storingPath := path.Join(IMAGE_DIR, folderHierarchy)
	os.MkdirAll(storingPath, os.ModePerm)

	// Create an empty file
	dst, err := os.Create(path.Join(storingPath, newFileName))
	defer dst.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy the uploaded file to the created file on the filesystem
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// re-arrange file name and folder hierarchy
	fmt.Fprintf(w, fmt.Sprintf("%d-%s%s", unixTime, dateStr, fileExtension))
}

func fetchImage(w http.ResponseWriter, r *http.Request) {
	imageName := chi.URLParam(r, "name")
	args := strings.Split(imageName, "-")

	//args[0] should be file id with `/image/` suffix
	//args[1] should be folder hierarchy with file extension
	if len(args) != 2 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	//splitting args[1] to get folder hierarchy and file extension
	splittedArg1 := strings.Split(args[1], ".")
	if len(splittedArg1) != 2 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	folderHierarchy := fmt.Sprintf("%s/%s/%s", splittedArg1[0][0:4], splittedArg1[0][4:6], splittedArg1[0][6:8])
	fileName := fmt.Sprintf("%s.%s", args[0], splittedArg1[1])
	filePath := path.Join(IMAGE_DIR, folderHierarchy, fileName)

	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, filePath)
}

func customNotfound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	return
}

func seed() {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	uploadDir := path.Join(workingDir, IMAGE_DIR)
	os.MkdirAll(uploadDir, os.ModePerm)
}

func main() {

	fmt.Println("Image server is starting")

	router := chi.NewRouter()

	router.Post("/upload/image", uploadImage)
	router.Get("/images/{name}", fetchImage)
	router.NotFound(customNotfound)

	http.ListenAndServe(":8080", router)
}
