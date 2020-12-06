package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	var (
		port             int
		wwwroot          string
		servedir         bool
		cachettl         string
		cachesize        string
		maxcacheitemsize string
		compress         bool
	)

	wd, err := os.Getwd()
	exitError(err)

	flag.BoolVar(&servedir, "servedir", false, "Enable serving directories")
	flag.IntVar(&port, "port", 6969, "listen port")
	flag.StringVar(&wwwroot, "root", wd, "serve root directory (defaults to current directory)")
	flag.StringVar(&cachettl, "cache-ttl", "5s", "cache time to live (default 5s)")
	flag.StringVar(&cachesize, "cache-size", "1gb", "cache size in bytes (default 1gb)")
	flag.StringVar(&maxcacheitemsize, "max-cache-item-size", "5mb", "max size of item to put in cache in bytes (default 5mb)")
	flag.BoolVar(&compress, "compress", false, "enable compression (default false)")
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

		s, err := NewSimpleServ(wwwroot, servedir, compress, cttl, parseKbMbGb(cachesize), parseKbMbGb(maxcacheitemsize))
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

func parseKbMbGb(val string) int {
	mul := 1

	switch {
	case strings.HasSuffix(val, "kb"):
		val = strings.TrimSuffix(val, "kb")
		mul = 1024
	case strings.HasSuffix(val, "mb"):
		val = strings.TrimSuffix(val, "mb")
		mul = 1024 * 1024
	case strings.HasSuffix(val, "gb"):
		val = strings.TrimSuffix(val, "gb")
		mul = 1024 * 1024 * 1024
	}

	sz, _ := strconv.ParseInt(val, 10, 32)
	return int(sz) * mul
}

func exitError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
