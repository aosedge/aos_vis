module github.com/aosedge/aos_vis

go 1.22.0

replace github.com/ThalesIgnite/crypto11 => github.com/aosedge/crypto11 v1.0.3-0.20220217163524-ddd0ace39e6f

require (
	github.com/aosedge/aos_common v0.0.0-20241230085333-54d55844bc38
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf
	github.com/gorilla/websocket v1.5.3
	github.com/sirupsen/logrus v1.9.3
	google.golang.org/grpc v1.69.0
)

require (
	github.com/ThalesIgnite/crypto11 v0.0.0-00010101000000-000000000000 // indirect
	github.com/google/go-tpm v0.9.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/miekg/pkcs11 v1.0.3-0.20190429190417-a667d056470f // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stefanberger/go-pkcs11uri v0.0.0-20230803200340-78284954bff6 // indirect
	github.com/thales-e-security/pool v0.0.2 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
	google.golang.org/protobuf v1.36.0 // indirect
)
