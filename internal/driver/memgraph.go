package driver

import (
	"context"
	"fmt"
	"log"
	
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type MemgraphDriver struct {
	Driver neo4j.DriverWithContext
}

func NewMemgraphDriver(uri, username, password string) (*MemgraphDriver, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, err
	}
	
	if err := driver.VerifyConnectivity(context.Background()); err != nil {
		return nil, err
	}
	
	log.Println("Connected to Memgraph")
	return &MemgraphDriver{Driver: driver}, nil
}

func (d *MemgraphDriver) Close(ctx context.Context) error {
	return d.Driver.Close(ctx)
}

func (d *MemgraphDriver) ExecuteQuery(ctx context.Context, query string, params map[string]interface{}) (neo4j.EagerResult, error) {
	result, err := neo4j.ExecuteQuery(ctx, d.Driver, query, params, neo4j.EagerResultTransformer)
	if err != nil {
		return neo4j.EagerResult{}, fmt.Errorf("failed to execute query: %w", err)
	}
	return *result, nil
}

func (d *MemgraphDriver) BuildIndices(ctx context.Context) error {
	// Basic constraints and indices for Graphiti
	// Memgraph supports Cypher index creation
	
	queries := []string{
		"CREATE INDEX ON :Entity(uuid);",
		"CREATE INDEX ON :Episodic(uuid);",
		"CREATE INDEX ON :Community(uuid);",
		"CREATE INDEX ON :Saga(uuid);",
		
		"CREATE INDEX ON :Entity(group_id);",
		"CREATE INDEX ON :Episodic(group_id);",
		"CREATE INDEX ON :Community(group_id);",
		"CREATE INDEX ON :Saga(group_id);",

		// Text indices? Memgraph has text search but syntax might differ from Neo4j fulltext
		// For now, simpler indices.
		
		// Vector indices setup would go here if using Memgraph's vector search capabilities
		// Example: CALL vector_search.create_index("Entity", "name_embedding", 1536, "COSINE");
		// Need to verify if Memgraph Mage is running with vector modules.
	}

	for _, q := range queries {
		_, err := d.ExecuteQuery(ctx, q, nil)
		if err != nil {
			log.Printf("Warning: failed to create index '%s': %v", q, err)
			// Continue, as index might already exist
		}
	}
	
	return nil
}
