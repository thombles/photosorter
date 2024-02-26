package main

import (
	"bufio"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

var sourcePath string
var targetPath string

func getSeenPath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatal(err, "No cache directory available")
	}
	return path.Join(cacheDir, "photosorter", "seen")
}

func prepareStateDir() {
	parent := path.Dir(getSeenPath())
	log.Println("Storing seen cache in:", parent)
	os.MkdirAll(parent, 0700)
}

func getSeen() map[string]bool {
	seenFiles := make(map[string]bool)
	file, err := os.Open(getSeenPath())
	if err != nil {
		return seenFiles
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		seenFiles[scanner.Text()] = true
	}
	return seenFiles
}

func setSeen(seenFiles map[string]bool) {
	file, err := os.Create(getSeenPath())
	if err != nil {
		return
	}
	defer file.Close()
	w := bufio.NewWriter(file)
	for key := range seenFiles {
		w.WriteString(key)
		w.WriteString("\n")
	}
	w.Flush()
}

func processFile(name string, modified time.Time) {
	sourceFile := path.Join(sourcePath, name)
	targetDir := path.Join(targetPath, strconv.Itoa(modified.Year()))
	targetFile := path.Join(targetDir, name)
	if _, err := os.Stat(targetFile); errors.Is(err, os.ErrNotExist) {
		log.Println("Copying src:", sourceFile, "target:", targetFile)
		r, err := os.Open(sourceFile)
		if err != nil {
			return
		}
		defer r.Close()
		os.MkdirAll(targetDir, 0700)
		w, err := os.Create(targetFile)
		if err != nil {
			return
		}
		defer w.Close()
		_, err = w.ReadFrom(r)
		if err != nil {
			// Try to clean up for next time
			os.Remove(targetFile)
		}
	} else {
		log.Println("Skipping because it exists, src:", sourceFile, "target:", targetFile)
	}
}

func doSort() {
	log.Println("Starting sort...")
	didProcess := false
	seenFiles := getSeen()
	newSeenFiles := make(map[string]bool)
	files, err := os.ReadDir(sourcePath)
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") {
			continue
		}
		if !file.Type().IsRegular() {
			continue
		}
		info, err := file.Info()
		if err != nil {
			continue
		}
		newSeenFiles[file.Name()] = true
		if seenFiles[file.Name()] {
			continue
		}
		processFile(file.Name(), info.ModTime())
		didProcess = true
	}
	setSeen(newSeenFiles)
	if didProcess {
		log.Println("Sort complete")
	} else {
		log.Println("Sort complete (no changes)")
	}
}

func sortWorker(changeTimes chan time.Time) {
	// Handle signals cleanly so we avoid stopping mid-execution
	stopping := make(chan os.Signal)
	signal.Notify(stopping, syscall.SIGTERM, syscall.SIGHUP)

	// Initial sort to catch any changes before startup
	lastRun := time.Now()
	doSort()
	for {
		select {
		case <-stopping:
			log.Println("Stopping due to signal")
			os.Exit(0)
		case t := <-changeTimes:
			if t.After(lastRun) {
				log.Println("Source directory did change")
				lastRun = time.Now()
				doSort()
			}
		}
	}
}

func watcher(changeTimes chan time.Time) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()
	w.Add(sourcePath)
	for range w.Events {
		changeTimes <- time.Now()
	}
}

func main() {
	const req = "REQUIRED"
	flag.StringVar(&sourcePath, "s", req, "source directory where photos are uploaded")
	flag.StringVar(&targetPath, "t", req, "target directory in which year directories are created")
	flag.Parse()
	if sourcePath == req || targetPath == req {
		log.Println("Source and Target paths must both be provided")
		os.Exit(1)
	}
	prepareStateDir()
	changeTimes := make(chan time.Time, 1024)
	go sortWorker(changeTimes)
	watcher(changeTimes)
}
