module github.com/NP-compete/arcana/pkg/db

go 1.23

require (
	github.com/NP-compete/arcana/pkg/logger v0.0.0
	github.com/lib/pq v1.12.3
)

replace github.com/NP-compete/arcana/pkg/logger => ../logger
