module codejudge/problems-service

go 1.19

require (
	codejudge/common v0.0.0
	github.com/go-chi/chi/v5 v5.0.11
	github.com/lib/pq v1.10.9
	go.uber.org/zap v1.27.0
)

require go.uber.org/multierr v1.10.0 // indirect

replace codejudge/common => ../common-go
