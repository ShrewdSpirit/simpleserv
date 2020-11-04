package main

import (
    "fmt"
    "io"
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
    WWWRoot string
}

func (s *SimpleServe) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    path := strings.Trim(req.URL.Path, "/")
    if path == "" {
        path = "index.html"
    }

    path = filepath.Join(s.WWWRoot, path)

    stat, err := os.Stat(path)
    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        s.log(nil, req, LogTypeErr, err.Error())
        return
    }

    if stat.IsDir() {
        w.WriteHeader(http.StatusNotImplemented)
    } else {
        file, err := os.Open(path)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            s.log(w, req, LogTypeErr, err.Error())
            return
        }
        defer file.Close()

        w.Header().Set("Content-Type", mime.TypeByExtension(path))
        w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))

        if _, err := io.Copy(w, file); err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            s.log(w, req, LogTypeErr, err.Error())
            return
        }
    }
}

func (s *SimpleServe) log(to http.ResponseWriter, req *http.Request, logtype LogType, msg string) {
    text := fmt.Sprintf("[%s] %s\n\t%s", logtype, msg, req.URL.String())
    fmt.Println(text)
    if to != nil {
        to.Write([]byte(text))
    }
}
