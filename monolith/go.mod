module codejudge/monolith

go 1.23.0

require (
	codejudge/common v0.0.0
	github.com/agext/levenshtein v1.2.3
	github.com/go-chi/chi/v5 v5.2.3
	github.com/go-chi/cors v1.2.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/golang-jwt/jwt/v5 v5.0.0
	github.com/goplus/llcppg v0.7.6
	github.com/lib/pq v1.10.9
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.13.0
)

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-farm v0.0.0-20240924180020-3414d57e47da // indirect
	github.com/dgryski/go-minhash v0.0.0-20190315135803-ad340ca03076 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/ekzhu/minhash-lsh v0.0.0-20190924033628-faac2c6342f8 // indirect
	github.com/goplus/llgo v0.11.6-0.20250824004317-e4218f90d792 // indirect
	go.uber.org/multierr v1.10.0 // indirect
)

replace codejudge/common => ../common
