package driver

import (
	"context"
	
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type GraphDriver interface {
	ExecuteQuery(ctx context.Context, query string, params map[string]interface{}) (neo4j.EagerResult, error)
	BuildIndices(ctx context.Context) error
	Close(ctx context.Context) error
}
