package main

import (
	"encoding/json"
	//"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
)

const (
	INDEX = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta http-equiv="X-UA-Compatible" content="ie=edge" />
    <title>File Upload</title>
  </head>
  <body>
    <h1>Data Exchange</h1>
    <ul>
      <li><a href="/upload">Upload</a></li>
      <li><a href="/files">Download</a></li>
      <li><a href="/text">Send Text</a></li>
    </ul>
</body> </html>`

	UPLOAD = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta http-equiv="X-UA-Compatible" content="ie=edge" />
    <title>File Upload</title>
  </head>
  <body>
    <h1>Upload</h1>
    <form enctype="multipart/form-data" action="/upload" method="post">
      <input type="file" name="myFile" /><br />
      <input type="submit" value="upload" />
    </form>
  </body>
</html>`

	SENDTEXT = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta http-equiv="X-UA-Compatible" content="ie=edge" />
    <title>Send Text</title>
  </head>
  <body>
    <h1>Send Text</h1>
    <form action="/text" method="post">
      <textarea name="textdata" rows="4" cols="50"></textarea><br />
      <input type="submit" value="send" />
    </form>
  </body>
</html>`

	CERTCMDS = `openssl ecparam -genkey -name secp384r1 -out server.key
openssl req -new -x509 -sha384 -key server.key -out server.crt -days 730`

	CONFIG_FILE = "./config.json"
)

var (
	config = Config{}
)

type Config struct {
	HostPort          string
	UploadFolder      string
	DownloadFolder    string
	BasicAuthUsername string
	BasicAuthPassword string
	BasicAuthRealm    string
	ServerCert        string
	ServerKey         string
	FileUploadSizeMB  int64
}

func readConfigFile(filename string) error {
	file, err := os.Open(filename)

	if err != nil {
		return err
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)

	if err != nil {
		return err
	}

	return nil
}

func showIPs() {
	if len(config.HostPort) > 0 && string(config.HostPort[0]) == ":" {
		fmt.Println("[*] Listening on the following interfaces")
	} else {
		fmt.Println("[*] Interface Information")
	}

	ifaces, err := net.Interfaces()

	if err != nil {
		fmt.Println(err)
		return
	}

	for _, i := range ifaces {
		fmt.Println("  ", i.Name+":")
		addrs, err := i.Addrs()

		if err != nil {
			fmt.Println(err)
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			fmt.Println("\t-", ip)
		}
	}
}

func sendText(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		text := r.FormValue("textdata")
		fmt.Println("\n[*] Text:\n" + text + "\n[*] ========= \n")
	}

	fmt.Fprintf(w, SENDTEXT)
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		fmt.Fprintf(w, UPLOAD)
		return
	}

	r.ParseMultipartForm(config.FileUploadSizeMB << 20)

	file, handler, err := r.FormFile("myFile")
	if err != nil {
		log.Println("Error Retrieving the File")
		log.Println(err)
		return
	}
	defer file.Close()

	log.Printf("Uploaded File: %+v\n", handler.Filename)
	log.Printf("File Size: %+v\n", handler.Size)
	log.Printf("MIME Header: %+v\n", handler.Header)

	tempFile, err := ioutil.TempFile(config.UploadFolder, "*-"+handler.Filename)
	if err != nil {
		log.Println(err)
	}
	defer tempFile.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
	}

	tempFile.Write(fileBytes)
	fmt.Fprintf(w, "Successfully Uploaded File\n")
}

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, INDEX)
}

func basicAuthHandler(h http.Handler) http.Handler {
	return basicAuthHandlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}))
}

func basicAuthHandlerFunc(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()

		if !ok || user != config.BasicAuthUsername || pass != config.BasicAuthPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+config.BasicAuthRealm+`"`)
			w.WriteHeader(401)
			w.Write([]byte("Unauthorised.\n"))
			return
		}

		handler(w, r)
	}
}

func createDir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0755)
	}
}

func setupRoutes() {

	fs := http.FileServer(http.Dir(config.DownloadFolder))
	http.Handle("/files/", http.StripPrefix("/files", basicAuthHandler(fs)))

	http.HandleFunc("/upload", basicAuthHandlerFunc(uploadFile))
	http.HandleFunc("/", basicAuthHandlerFunc(index))
	http.HandleFunc("/text", basicAuthHandlerFunc(sendText))

	fmt.Println("[*] Listining on", config.HostPort)
	err := http.ListenAndServeTLS(config.HostPort, config.ServerCert, config.ServerKey, nil)

	if err != nil {
		log.Fatal("ListenAndServe: ", err)
		//fmt.Println("[!] If you have no server.crt and server.key, run the following commands:\n\n", CERTCMDS)
	}
}

func checkIfFileExists(filePath, errMessage string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Println("[!] File", filePath, "does not exist!\n   ", errMessage)
		os.Exit(1)
	}
}

func readConfig() {
	checkIfFileExists(CONFIG_FILE, "Please create a configureation file named "+CONFIG_FILE)

	err := readConfigFile(CONFIG_FILE)

	if err != nil {
		fmt.Println("")
		log.Println(err)
		os.Exit(1)
	}
}

func main() {
	readConfig()

	showIPs()

	createDir(config.UploadFolder)
	createDir(config.DownloadFolder)

	checkIfFileExists(config.ServerKey, config.ServerKey+" is missing. Please generate the file via:\n"+CERTCMDS)
	checkIfFileExists(config.ServerCert, config.ServerCert+" is missing. Please generate the file via:\n"+CERTCMDS)

	setupRoutes()
}
