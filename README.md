# RPS-Offline
Builds an executable that will download the latest version (rpsmaster.zip) from github as a zip and start a local webserver

If a zip is present, it will not download anything so you can use it offline

Adds a customs script to export to csv or json
* Push (j) to export to (j)son
* Push (c) to export to (c)sv
* Push (q) to stop the process

## Arguments
Usage of c:\rps-offline\rps-offline.exe:
*  -bind string
    *        Name for binding, keep localhost for extra security (default "localhost")
*  -browse
    *        Start explorer with url (default true)
*  -indexdir string
    *        Directory in zip which contains index.html (base directory) (default "rps-master")
*  -port int
    *        Port for local web server (default 17132)
*  -postscript string
    *        Run after json export, if empty, nothing is done
*  -scriptlang string
    *        Select scripting language (default "cmd") [cmd,powershell]
*  -srcurl string
    *        Source url to zip (default "https://github.com/tdewin/rps/archive/master.zip")
*  -tgtzip string
    *        Location where zip is download and where it will be used, alternatively can be used to point to another zip (default "rpsmaster.zip")