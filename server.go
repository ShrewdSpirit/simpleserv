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

	stat, err := os.Stat(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.log(nil, req, LogTypeErr, err.Error())
		return
	}

	if stat.IsDir() {
		ignoreIndexStr := req.URL.Query().Get("ignore-index")
		ignoreIndex, _ := strconv.ParseBool(ignoreIndexStr)
		if err := s.serveDir(w, path, ignoreIndex); err != nil {
			s.log(nil, req, LogTypeErr, err.Error())
		}

		return
	}

	if err := s.serveFile(w, path, stat.Size()); err != nil {
		s.log(nil, req, LogTypeErr, err.Error())
		return
	}
}

func (s *SimpleServe) serveDir(w http.ResponseWriter, path string, ignoreIndex bool) error {
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

	ls, err := ioutil.ReadDir(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	responseBytes, err := json.Marshal(ls)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

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

	w.Header().Set("Content-Type", mime.TypeByExtension(path))
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
