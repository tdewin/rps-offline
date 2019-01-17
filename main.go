package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

func main() {

	masterpack := flag.String("srcurl", "https://github.com/tdewin/rps/archive/master.zip", "Source url to zip")
	targetzip := flag.String("tgtzip", "rpsmaster.zip", "Location where zip is download and where it will be used, alternatively can be used to point to another zip")
	subdirinzip := flag.String("indexdir", "rps-master", "Directory in zip which contains index.html (base directory)")
	browse := flag.Bool("browse", true, "Start explorer with url")
	port := flag.Int("port", 17132, "Port for local web server")
	bindto := flag.String("bind", "localhost", "Name for binding, keep localhost for extra security")
	postscript := flag.String("postscript", "", "Run after json export, if empty, nothing is done")
	scriptlang := flag.String("scriptlang", "cmd", "Select scripting language. Can be cmd or powershell on windows")

	flag.Parse()

	if _, err := os.Stat(*targetzip); os.IsNotExist(err) {
		out, err := os.Create(*targetzip)
		if err != nil {
			log.Println("Can not open target for zip location ", *targetzip)
			log.Fatal(err)
		}
		defer out.Close()

		resp, err := http.Get(*masterpack)
		if err != nil {
			log.Println("Can not download masterpack zip from ", *masterpack)
			log.Fatal(err)
		}
		defer resp.Body.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			log.Println("Problem copying from download")
			log.Fatal(err)
		}

	}

	if _, err := os.Stat(*targetzip); os.IsNotExist(err) {
		log.Printf("Can not find zip")
		log.Fatal(err)
	} else {
		zipr, err := zip.OpenReader(*targetzip)
		defer zipr.Close()
		if err != nil {
			log.Printf("Can not open zip")
			log.Fatal(err)
		}

		/*for _, f := range zipr.File {
			log.Printf("file %s", f.Name)
		}*/

		stop := false
		listenTo := fmt.Sprintf("%s:%d", *bindto, *port)
		log.Println("Starting ", listenTo)

		srv := &http.Server{Addr: listenTo}
		http.Handle("/", ZipHandler{zipdata: zipr, self: srv, stop: &stop, subdirinzip: *subdirinzip, postscript: *postscript, scriptlang: *scriptlang})

		go func() {
			srv.ListenAndServe()
		}()

		if *browse {
			go func() {
				time.Sleep(2 * time.Second)
				if runtime.GOOS == "windows" {
					cmd := exec.Command("explorer", fmt.Sprintf("http://%s", listenTo))
					lerr := cmd.Start()
					if lerr != nil {
						log.Printf("%v", lerr)
					}
				}
			}()
		}

		for !(stop) {
			time.Sleep(1 * time.Second)
		}
		log.Printf("Shutting down gracefully")
		time.Sleep(3 * time.Second)
		srv.Shutdown(nil)

	}

}
