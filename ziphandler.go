package main

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"time"
)

//ZipHandler is main struct so that the same zipdata is always used
type ZipHandler struct {
	zipdata     *zip.ReadCloser
	self        *http.Server
	subdirinzip string
	stop        chan bool
	postscript  string
	scriptlang  string
}

//Adding some extra code to pureengine.js to add some functionality when you click (q)uit or (c)sv
//idea is to add this later to a script called offline.js . When the default is use, it should just include function isOffline() { return false }
//when offline version is used, this function is used instead of file
func (z ZipHandler) WriteJSAddon(rw http.ResponseWriter) {
	fmt.Fprint(rw, `

function isOffline() {
	return true
}

$(document).ready(
	function() {
		$( document ).keypress(function( event ) {
			if (event.which == 99) {
				var gsv = GUIStateVariables()
				var files = gsv.backupResult.getWorstCaseFiles()
				var sep = ","
				var text = [("#,\""+getExportStr())+"\""]

				var fname = "rpsout-"+moment().format("YYYYMMDDHHmmss")+".csv"
				var qname = window.prompt("Define CSV export name",fname);
				if (qname != null) {
					if (qname != "") {
						fname = qname
					}

					//https://github.com/tdewin/rps/blob/master/pureengine.js 243 for inspiration
					for(var counter=0;counter < files.length;counter = counter + 1 ) {
						var file = files[counter];
						var stats = file.getDataStats()
						var textline = [file.toSimpleRetentionString(),file.fullfile(),stats.f()]
						text.push(textline.join(sep))
					}
					textline = ["","Workspace",gsv.backupResult.worstCaseDayWorkingSpace]
					text.push(textline.join(sep))
					
					headersobj = {"X-Action": "savecsv","X-Savefile":fname}
					

					$.ajax({
						type: "POST",
						url: "/offlinehandler",
						data: text.join("\r\n"),
						headers: headersobj,
						success: function (data) {
							alert("Exported to CSV : "+data)
						},
						dataType: "text"
					});
				}
			} else if (event.which == 106)  {
				var gsv = GUIStateVariables()
				var br = gsv.backupResult
				var files = gsv.backupResult.getWorstCaseFiles()
				var datestamp = moment().format("YYYYMMDDHHmmss")


				var fname = "rpsout-"+datestamp+".json"

				var qname = window.prompt("Define JSON export name",fname);
				if (qname != null) {
					if (qname != "") {
						fname = qname
					}
					jsondata = { srcurl:getExportStr(),datestamp:datestamp,origfname:qname}

					var filesarr = []
					for(var counter=0;counter < files.length;counter = counter + 1 ) {
						var file = files[counter];
						var stats = file.getDataStats()
						var jfile = {
							"retention":file.toSimpleRetentionString(),
							"backuptype":file.type,
							"filename":file.fullfile(),
							"filesize":stats.f(),
							"source":stats.s(),
							"sourcedelta":stats.sd(),
							"compression":stats.c(),
							"transferCompression":stats.t(),
							"delta":stats.d(),
						}
						filesarr.push(jfile)
					}
					jsondata.workspace = br.worstCaseDayWorkingSpace
					jsondata.size = br.worstCaseSize
					jsondata.sizeandworkspace = br.worstCaseSizeWithWorkingSpace

					jsondata.files = filesarr

					headersobj = {"X-Action": "savejson","X-Savefile":fname}


					$("#overlay").show();
					var opts = {
						lines: 13 // The number of lines to draw
						, length: 28 // The length of each line
						, width: 14 // The line thickness
						, radius: 42 // The radius of the inner circle
						, scale: 1 // Scales overall size of the spinner
						, corners: 1 // Corner roundness (0..1)
						, color: '#000' // #rgb or #rrggbb or array of colors
						, opacity: 0.25 // Opacity of the lines
						, rotate: 0 // The rotation offset
						, direction: 1 // 1: clockwise, -1: counterclockwise
						, speed: 1 // Rounds per second
						, trail: 60 // Afterglow percentage
						, fps: 20 // Frames per second when using setTimeout() as a fallback for CSS
						, zIndex: 2e9 // The z-index (defaults to 2000000000)
						, className: 'spinner' // The CSS class to assign to the spinner
						, top: '50%' // Top position relative to parent
						, left: '50%' // Left position relative to parent
						, shadow: false // Whether to render a shadow
						, hwaccel: false // Whether to use hardware acceleration
						, position: 'absolute' // Element positioning
						}
					var target = document.getElementById('overlay')
					var spinner = new Spinner(opts).spin(target);

					$.ajax({
						type: "POST",
						url: "/offlinehandler",
						data: JSON.stringify(jsondata),
						headers: headersobj,
						dataType: "text"
					}).done(function(data) {
						alert( data);
					}).fail(function() {
						alert( "error" );
					}).always(function() {
						spinner.stop()
						$("#overlay").hide();
					});
				  }
			} else if (event.which == 113)  {
				$.ajax({
					type: "GET",
					url: "/offlinehandler",
					headers: {"X-Action": "stop"},	
					success: function(data) {
						alert("Stopping")
					}				
				})
			} else if (event.which == 79) {
				window.location.href = "/?m=3&s=102400&r=5&c=50&d=10&i=D&dgr=10&dgy=1&dg=0&re=1&gaf=0&g=6,12,8,3&e"
			}
		});
	}
)
		`)
}

func (z ZipHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	path := req.URL.Path[1:]

	if path == "" {
		path = "index.html"
	}
	if path != "offlinehandler" {
		jpath := fmt.Sprintf("%s/%s", z.subdirinzip, path)
		found := false
		for _, f := range z.zipdata.File {
			if f.Name == jpath {
				found = true
				rf, _ := f.Open()
				defer rf.Close()
				webfile, _ := ioutil.ReadAll(rf)
				rw.Write(webfile)

				if path == "pureengine.js" {
					z.WriteJSAddon(rw)
				}
			}

		}
		if !found {
			fmt.Fprintf(rw, "Could not find %s", jpath)
		}
	} else {
		action := req.Header["X-Action"]

		fnames := req.Header["X-Savefile"]

		/*for n, h := range req.Header {
			for i, sh := range h {
				log.Printf("%s %d %s", n, i, sh)
			}
		}*/
		if len(action) > 0 {
			switch action[0] {
			case "savecsv":
				{
					body, err := ioutil.ReadAll(req.Body)
					if err == nil {
						t := time.Now()

						fname := fmt.Sprintf("%s.csv", t.Format("2006-01-02-15-04-05"))
						if len(fnames) > 0 && fnames[0] != "" {
							fname = fnames[0]
						}

						ioutil.WriteFile(fname, body, 0644)
						log.Printf("Outputing to %s", fname)
						fmt.Fprint(rw, fname)
					} else {
						fmt.Fprint(rw, "Something went wrong")
						log.Print(err)
					}
				}
			case "savejson":
				{
					body, err := ioutil.ReadAll(req.Body)
					if err == nil {
						t := time.Now()

						fname := fmt.Sprintf("%s.json", t.Format("2006-01-02-15-04-05"))
						if len(fnames) > 0 && fnames[0] != "" {
							fname = fnames[0]
						}
						abspath, err := filepath.Abs(fname)

						if err != nil {
							fmt.Fprint(rw, "Something went wrong")
							log.Print(err)
						} else {
							ioutil.WriteFile(abspath, body, 0644)
							if z.postscript != "" {

								switch z.scriptlang {
								case "cmd":
									{
										cmd := exec.Command(z.postscript, abspath)
										log.Println("Running script ", z.postscript, abspath)
										stdoutStderr, err := cmd.CombinedOutput()
										if err != nil {
											log.Printf("%v", err)
										}
										log.Printf("%s", stdoutStderr)
									}
								case "powershell":
									{
										cmd := exec.Command("powershell.exe", "-file", z.postscript, abspath)
										log.Println("Running script ", "powershell.exe", "-file", z.postscript, abspath)
										stdoutStderr, err := cmd.CombinedOutput()
										if err != nil {
											log.Printf("%v", err)
										}
										log.Printf("%s", stdoutStderr)
									}
								}

							}
							log.Printf("Outputing to %s", fname)
							fmt.Fprint(rw, fname)
						}
					} else {
						fmt.Fprint(rw, "Something went wrong")
						log.Print(err)
					}
				}
			case "stop":
				{
					fmt.Fprint(rw, "Quiting")

					z.stop <- true
				}
			}

		} else {
			fmt.Fprint(rw, "no action defined")
		}

	}

}
