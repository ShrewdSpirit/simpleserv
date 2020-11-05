package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type LogType string

const (
	LogTypeInfo LogType = "Info"
	LogTypeErr  LogType = "Error"
)

type SimpleServe struct {
	WWWRoot  string
	ServeDir bool
}

func (s *SimpleServe) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := filepath.Join(s.WWWRoot, strings.Trim(req.URL.Path, "/"))

	pretyStr := req.URL.Query().Get("prety")
	prety, _ := strconv.ParseBool(pretyStr)

	stat, err := os.Stat(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.log(nil, req, LogTypeErr, err.Error())
		return
	}

	if stat.IsDir() {
		ignoreIndexStr := req.URL.Query().Get("ignore-index")
		ignoreIndex, _ := strconv.ParseBool(ignoreIndexStr)
		if err := s.serveDir(w, req, path, ignoreIndex, prety); err != nil {
			s.log(nil, req, LogTypeErr, err.Error())
		}

		return
	}

	if err := s.serveFile(w, path, stat.Size()); err != nil {
		s.log(nil, req, LogTypeErr, err.Error())
		return
	}
}

func (s *SimpleServe) serveDir(w http.ResponseWriter, req *http.Request, path string, ignoreIndex, prety bool) error {
	if !ignoreIndex {
		indexPath := filepath.Join(path, "index.html")
		if stat, err := os.Stat(indexPath); !os.IsNotExist(err) && !stat.IsDir() {
			return s.serveFile(w, indexPath, stat.Size())
		}
	}

	if !s.ServeDir {
		w.WriteHeader(http.StatusUnauthorized)
		return nil
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	ls := make([]struct {
		Name    string
		Size    int64
		ModTime int64
		IsDir   bool
		Link    string
	}, len(files))
	for i, fileInfo := range files {
		ls[i].Name = fileInfo.Name()
		ls[i].Size = fileInfo.Size()
		ls[i].ModTime = fileInfo.ModTime().UnixNano()
		ls[i].IsDir = fileInfo.IsDir()

		scheme := req.URL.Scheme
		if scheme == "" {
			scheme = "http"
		}
		ls[i].Link = fmt.Sprintf("%s://%s%s", scheme, req.Host, filepath.Join(req.URL.Path, fileInfo.Name()))
	}

	var responseBytes []byte
	if prety {
		responseBytes, err = json.MarshalIndent(ls, "", "  ")
	} else {
		responseBytes, err = json.Marshal(ls)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBytes)

	return nil
}

func (s *SimpleServe) serveFile(w http.ResponseWriter, path string, length int64) error {
	file, err := os.Open(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	defer file.Close()

	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(path)))
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))

	if _, err := io.Copy(w, file); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	return nil
}

func (s *SimpleServe) log(to http.ResponseWriter, req *http.Request, logtype LogType, msg string) {
	text := fmt.Sprintf("[%s] [at %s] %s", logtype, req.URL.String(), msg)
	log.Println(text)
	if to != nil {
		to.Write([]byte(text))
	}
}
