package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	flagCorpus  = flag.String("corpus", "", "dir with input corpus (one file per input)")
	flagWorkdir = flag.String("workdir", "", "dir with persistent work data")
	flagProcs   = flag.Int("procs", runtime.NumCPU(), "parallelism level")
	flagTimeout = flag.Int("timeout", 5000, "test timeout, in ms")
	flagMaster  = flag.String("master", "", "master mode (value is master address)")
	flagSlave   = flag.String("slave", "", "slave mode (value is master address)")
	flagBin     = flag.String("bin", "", "test binary built with go-fuzz-build")
	flagV       = flag.Bool("v", false, "verbose mode")

	shutdown  uint32
	shutdownC = make(chan struct{})
)

func main() {
	flag.Parse()
	if *flagCorpus == "" {
		log.Fatalf("-corpus is not set")
	}
	if *flagWorkdir == "" {
		log.Fatalf("-workdir is not set")
	}
	if *flagBin == "" {
		log.Fatalf("-bin is not set")
	}
	if *flagMaster != "" && *flagSlave != "" {
		log.Fatalf("both -master and -slave are specified")
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT)
		<-c
		atomic.StoreUint32(&shutdown, 1)
		close(shutdownC)
		log.Printf("shutting down...")
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	runtime.GOMAXPROCS(runtime.NumCPU())
	syscall.Setpriority(syscall.PRIO_PROCESS, 0, 19)

	if *flagMaster != "" || *flagSlave == "" {
		if *flagMaster == "" {
			*flagMaster = "localhost:0"
		}
		ln, err := net.Listen("tcp", *flagMaster)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		if *flagMaster == "localhost:0" && *flagSlave == "" {
			*flagSlave = ln.Addr().String()
		}
		go masterMain(ln)
	}

	if *flagSlave != "" {
		for i := 0; i < *flagProcs; i++ {
			go slaveMain()
		}
	}

	select {}
}
