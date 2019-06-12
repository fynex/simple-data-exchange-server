# simple-data-exchange-server

A pretty simple HTTPS data exchange server to upload and download data, which provides basic authentication on HTTPS encryption out of the box.

## Setup

### Compilation

Compilation is pretty simple. The method via git clone looks like this:

```
git clone https://github.com/fynex/simple-data-exchange-server
cd simple-data-exchange-server
go build
```

### Certificate

To create valid certificates openssl can be used. Here we create a certificate which is valid for two years:

```
openssl ecparam -genkey -name secp384r1 -out server.key
openssl req -new -x509 -sha384 -key server.key -out server.crt -days 730`
```

### Config File

To configure the server you can use the following json data:

```json
{
	  "HostPort"          : ":8000",
	  "UploadFolder"      : "upload",
	  "DownloadFolder"    : "download",
	  "BasicAuthUsername" : "USERNAME",
	  "BasicAuthPassword" : "PASSWORD",
	  "BasicAuthRealm"    : "A_REALM",
	  "ServerCert"       : "server.crt",
	  "ServerKey"        : "server.key",
	  "FileUploadSizeMB"  : 100
}
```

To bind the server to an specific interface set `"HostPort" : ":8000"` to `IP:PORT`. 

If you want to have the upload and download directory in the same directory set the `UploadFolder` equal to the `DownloadFolder` entry.
