package main

import (
	"flag"
	"fmt"
	"github.com/svip/sitestrapper/strapper"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	var inDir, outDir string
	flag.StringVar(&inDir, "i", "", "Input directory")
	flag.StringVar(&outDir, "o", "", "Output directory")
	var port int
	flag.IntVar(&port, "port", 0, "Port number to listen on (will start server if provided)")

	flag.Parse()

	if len(inDir) == 0 {
		log.Fatal("Input directory required")
	}
	if len(outDir) == 0 {
		log.Fatal("Output directory required")
	}

	log.Printf("Generating site from %s to %s...\n", inDir, outDir)
	ss := strapper.NewSiteStrapper(inDir, outDir)
	err := ss.GenerateSite()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	if port > 0 {
		log.Println("Serving", outDir)
		log.Printf("On http://localhost:%d/\n", port)
		http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
			log.Printf("%s %s", request.Method, request.URL)
			path := filepath.Join(outDir, request.URL.String())
			if info, err := os.Stat(path); err != nil {
				log.Println("Error", err)
				writer.WriteHeader(http.StatusNotFound)
				return
			} else if info.IsDir() {
				path = fmt.Sprintf("%s/index.html", path)
			}
			log.Println("Serving file", path)
			f, err := os.Open(path)
			if err != nil {
				log.Println("Error", err)
				writer.WriteHeader(http.StatusNotFound)
				return
			}
			defer f.Close()

			http.ServeContent(writer, request, filepath.Base(path), time.Time{}, f)
		})
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
	}
}
