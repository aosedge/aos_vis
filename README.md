## Vehicle Information Service

aos_vis is web socket server which provides vehicle information data using W3C protocol

https://www.w3.org/TR/vehicle-information-service/

Specify parameters in visconfig.json file:

```json
{
	"ServerURL": "localhost:8088",
	"VISCert": "data/wwwivi.crt.pem",
	"VISKey": "data/wwwivi.key.pem",
	"Adapters":[
		{
			"Name":"MessageAdapter"
		},{
			"Name":"SensorEmulatorAdapter",
			"Params": {
				"SensorURL":"http://sensors:8800"
			}
		},{
			"Name":"TestAdapter"
		}
	]
}
```

Put root certificate rootCA.crt.pem to /etc/ssl/certs in your system.