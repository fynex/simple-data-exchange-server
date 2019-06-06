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

	CERTCMDS = `openssl genrsa -out server.key 2048
openssl ecparam -genkey -name secp384r1 -out server.key`
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
	FileUploadSizeMB  int64
}

func readConfig(filename string) error {
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
	ifaces, err := net.Interfaces()

	if err != nil {
		fmt.Println(err)
		return
	}

	for _, i := range ifaces {
		fmt.Println("[*]", i.Name+":")
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

	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	r.ParseMultipartForm(config.FileUploadSizeMB << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
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

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	createDir(config.UploadFolder)

	tempFile, err := ioutil.TempFile(config.UploadFolder, "*-"+handler.Filename)
	if err != nil {
		log.Println(err)
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)
	// return that we have successfully uploaded our file!
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
		os.Mkdir(path, 755)
	}
}

func setupRoutes() {
	createDir(config.DownloadFolder)

	fs := http.FileServer(http.Dir(config.DownloadFolder))
	http.Handle("/files/", http.StripPrefix("/files", basicAuthHandler(fs)))

	http.HandleFunc("/upload", basicAuthHandlerFunc(uploadFile))
	http.HandleFunc("/", basicAuthHandlerFunc(index))
	http.HandleFunc("/text", basicAuthHandlerFunc(sendText))

	fmt.Println("[*] Listining on", config.HostPort)
	err := http.ListenAndServeTLS(config.HostPort, "server.crt", "server.key", nil)

	if err != nil {
		log.Fatal("ListenAndServe: ", err)
		fmt.Println("[!] If you have no server.crt and server.key, run the following commands:\n\n", CERTCMDS)
	}
}

func main() {
	showIPs()

	err := readConfig("./config.json")
	if err != nil {
		fmt.Println("")
		log.Println(err)
		os.Exit(-1)
	}

	setupRoutes()
}
