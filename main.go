package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"
)

func main() {
	var port int
	var wwwroot string
	var servedir bool
	var cachettl string
	var cachesize int
	var maxcacheitemsize int

	wd, err := os.Getwd()
	exitError(err)

	flag.BoolVar(&servedir, "servedir", false, "Enable serving directories")
	flag.IntVar(&port, "port", 6969, "listen port")
	flag.StringVar(&wwwroot, "root", wd, "serve root directory (defaults to current directory)")
	flag.StringVar(&cachettl, "cache-ttl", "10s", "cache time to live (default 0)")
	flag.IntVar(&cachesize, "cache-size", 1024*1024*50, "cache size in bytes (default 52428800)")
	flag.IntVar(&maxcacheitemsize, "max-cache-item-size", 1024*1024*5, "max size of item to put in cache in bytes (default 5242880)")
	flag.Parse()

	cpuProfFile := os.Getenv("CPUPROF")
	memProfFile := os.Getenv("MEMPROF")

	if cpuProfFile != "" {
		cpuFile, err := os.Create(cpuProfFile)
		if err != nil {
			fmt.Println(fmt.Errorf("could not create CPU profile: %w", err))
			return
		}
		defer cpuFile.Close()

		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			fmt.Println(fmt.Errorf("could not start CPU profile: %w", err))
			return
		}
		defer pprof.StopCPUProfile()
	}

	cttl, err := time.ParseDuration(cachettl)
	exitError(err)

	go func() {
		fmt.Printf("serving directory %s\n", wwwroot)
		fmt.Printf("listening on :%d\n", port)

		s, err := NewSimpleServ(wwwroot, servedir, cttl, cachesize, maxcacheitemsize)
		exitError(err)

		if err = http.ListenAndServe(fmt.Sprintf(":%d", port), s); err != nil {
			exitError(err)
		}
	}()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGINT)

	<-exit
	fmt.Println("exit")

	if memProfFile != "" {
		memFile, err := os.Create(memProfFile)
		if err != nil {
			fmt.Println(fmt.Errorf("could not create memory profile", err))
			return
		}
		defer memFile.Close()

		runtime.GC()
		if err := pprof.WriteHeapProfile(memFile); err != nil {
			fmt.Println(fmt.Errorf("could not write memory profile", err))
			return
		}
	}
}

func exitError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
