package driver

const (
	SaveEntityNodeQuery = `
		MERGE (n:Entity {uuid: $uuid})
		SET n.name = $name,
			n.group_id = $group_id,
			n.created_at = $created_at,
			n.summary = $summary,
			n.name_embedding = $name_embedding,
			n.attributes = $attributes
		WITH n
		FOREACH (label IN $labels | SET n:label)
		RETURN n.uuid AS uuid
	`

	SaveEpisodicNodeQuery = `
		MERGE (n:Episodic {uuid: $uuid})
		SET n.name = $name,
			n.group_id = $group_id,
			n.created_at = $created_at,
			n.valid_at = $valid_at,
			n.content = $content,
			n.source = $source,
			n.source_description = $source_description,
			n.entity_edges = $entity_edges
		RETURN n.uuid AS uuid
	`

	SaveCommunityNodeQuery = `
		MERGE (n:Community {uuid: $uuid})
		SET n.name = $name,
			n.group_id = $group_id,
			n.created_at = $created_at,
			n.summary = $summary,
			n.name_embedding = $name_embedding
		RETURN n.uuid AS uuid
	`
	
	SaveEntityEdgeQuery = `
		MATCH (source:Entity {uuid: $source_uuid})
		MATCH (target:Entity {uuid: $target_uuid})
		MERGE (source)-[e:RELATES_TO {uuid: $uuid}]->(target)
		SET e.name = $name,
			e.fact = $fact,
			e.group_id = $group_id,
			e.created_at = $created_at,
			e.expired_at = $expired_at,
			e.valid_at = $valid_at,
			e.invalid_at = $invalid_at,
			e.episodes = $episodes,
			e.fact_embedding = $fact_embedding,
			e.attributes = $attributes
		RETURN e.uuid AS uuid
	`

	SaveEpisodicEdgeQuery = `
		MATCH (episode:Episodic {uuid: $source_uuid})
		MATCH (node:Entity {uuid: $target_uuid})
		MERGE (episode)-[e:MENTIONS {uuid: $uuid}]->(node)
		SET e.group_id = $group_id,
			e.created_at = $created_at
		RETURN e.uuid AS uuid
	`
)
