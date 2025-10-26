module codejudge/monolith

go 1.23.0

require (
	codejudge/common v0.0.0
	github.com/dgryski/go-farm v0.0.0-20240924180020-3414d57e47da
	github.com/dgryski/go-minhash v0.0.0-20190315135803-ad340ca03076
	github.com/ekzhu/minhash-lsh v0.0.0-20190924033628-faac2c6342f8
	github.com/go-chi/chi/v5 v5.2.3
	github.com/go-chi/cors v1.2.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/goplus/llcppg v0.7.6
	github.com/lib/pq v1.10.9
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.35.0
)

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-metro v0.0.0-20250106013310-edb8663e5e33 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dgryski/go-spooky v0.0.0-20170606183049-ed3d087f40e2 // indirect
	github.com/goplus/llgo v0.11.6-0.20250824004317-e4218f90d792 // indirect
	go.uber.org/multierr v1.10.0 // indirect
)

replace codejudge/common => ../common
