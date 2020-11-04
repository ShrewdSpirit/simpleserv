package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var port int
	var wwwroot string
	var servedir bool

	wd, err := os.Getwd()
	exitError(err)

	flag.BoolVar(&servedir, "servedir", false, "Enable serving directories")
	flag.IntVar(&port, "port", 6969, "listen port")
	flag.StringVar(&wwwroot, "root", wd, "serve root directory (defaults to current directory)")
	flag.Parse()

	go func() {
		fmt.Printf("serving directory %s\n", wwwroot)
		fmt.Printf("listening on :%d\n", port)

		s := &SimpleServe{
			WWWRoot:  wwwroot,
			ServeDir: servedir,
		}

		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), s); err != nil {
			exitError(err)
		}
	}()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGINT)

	<-exit
	fmt.Println("exit")
}

func exitError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
