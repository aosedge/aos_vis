## Vehicle Information Service

aos_vis is web socket server which provides vehicle information data using W3C protocol

https://www.w3.org/TR/vehicle-information-service/

Specify path to certificates in visconfig.json file.  example is located in config/visconfig.json

```json
{
	"ServerUrl": "localhost:8088",
	"VISCert": "src/gitpct.epam.com/epmd-aepr/aos_vis/data/wwwivi.crt.pem",
	"VISKey": "src/gitpct.epam.com/epmd-aepr/aos_vis/data/wwwivi.key.pem"
}
```

Put root certificate rootCA.crt.pem to /etc/ssl/certs in your system.