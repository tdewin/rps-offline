package main

import (
	"archive/zip"
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

//open the window browser and go to the main page
func OpenBrowser(seconds time.Duration, listenTo string) {
	time.Sleep(seconds * time.Second)
	//linux etc default https://stackoverflow.com/questions/39320371/how-start-web-server-to-open-page-in-browser-in-golang
	cmd := exec.Command("xdg-open", fmt.Sprintf("http://%s", listenTo))
	switch runtime.GOOS {
	case "windows":
		{
			cmd = exec.Command("explorer", fmt.Sprintf("http://%s", listenTo))
		}
	case "darwin":
		{
			cmd = exec.Command("open", fmt.Sprintf("http://%s", listenTo))
		}
	}
	lerr := cmd.Start()
	if lerr != nil {
		log.Printf("%v", lerr)
	}
}
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
			log.Printf("Can not open zip, if it is corrupted, try deleting it so that it will download again")
			log.Fatal(err)
		}

		/*for _, f := range zipr.File {
			log.Printf("file %s", f.Name)
		}*/
		//stop handler so that a stop command can be given from the command line
		stop := make(chan bool)
		listenTo := fmt.Sprintf("%s:%d", *bindto, *port)
		log.Println("Starting ", listenTo)

		srv := &http.Server{Addr: listenTo}
		http.Handle("/", ZipHandler{zipdata: zipr, self: srv, stop: stop, subdirinzip: *subdirinzip, postscript: *postscript, scriptlang: *scriptlang})

		go func() {
			srv.ListenAndServe()
		}()

		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				switch strings.ToLower(scanner.Text()) {
				case "quit":
					stop <- true
				case "open":
					OpenBrowser(0, listenTo)
				default:
					log.Println("Supported commands : quit, open")
				}
			}

		}()

		if *browse {
			//opens the browser with 2 seconds of delay, should be more then enough to get the service started
			go OpenBrowser(2, listenTo)
		}

		//wait for stop to be true
		<-stop
		log.Printf("Shutting down gracefully")
		srv.Shutdown(context.Background())

	}

}
