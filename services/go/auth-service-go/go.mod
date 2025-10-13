module codejudge/auth-service-go

go 1.19

require (
	codejudge/common v0.0.0
	github.com/go-chi/chi/v5 v5.0.10
	github.com/go-chi/cors v1.2.1
	github.com/golang-jwt/jwt/v5 v5.0.0
	github.com/lib/pq v1.10.9
	go.uber.org/zap v1.25.0
	golang.org/x/crypto v0.13.0
)

require (
	go.uber.org/multierr v1.10.0 // indirect
)

replace codejudge/common => ../common-go