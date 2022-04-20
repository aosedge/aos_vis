module github.com/aoscloud/aos_vis

go 1.14

replace github.com/ThalesIgnite/crypto11 => github.com/aoscloud/crypto11 v1.0.3-0.20220217163524-ddd0ace39e6f

require (
	github.com/aoscloud/aos_common v0.0.0-20220519144115-e4d62a88d016
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f
	github.com/gorilla/websocket v1.4.2
	github.com/sirupsen/logrus v1.8.1
	google.golang.org/grpc v1.41.0
)
