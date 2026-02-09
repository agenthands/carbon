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

	SaveSagaNodeQuery = `
		MERGE (n:Saga {uuid: $uuid})
		SET n.name = $name,
			n.group_id = $group_id,
			n.created_at = $created_at
		RETURN n.uuid AS uuid
	`

	GetSagaByNameQuery = `
		MATCH (s:Saga {name: $name, group_id: $group_id})
		RETURN s.uuid as uuid, s.name as name, s.group_id as group_id, s.created_at as created_at
	`

	GetPreviousEpisodeInSagaQuery = `
		MATCH (s:Saga {uuid: $saga_uuid})-[:HAS_EPISODE]->(e:Episodic)
		WHERE e.uuid <> $current_episode_uuid
		RETURN e.uuid AS uuid
		ORDER BY e.valid_at DESC, e.created_at DESC
		LIMIT 1
	`
	
	SaveNextEpisodeEdgeQuery = `
		MATCH (source:Episodic {uuid: $source_uuid})
		MATCH (target:Episodic {uuid: $target_uuid})
		MERGE (source)-[e:NEXT_EPISODE {uuid: $uuid}]->(target)
		SET e.group_id = $group_id,
			e.created_at = $created_at
		RETURN e.uuid AS uuid
	`

	SaveHasEpisodeEdgeQuery = `
		MATCH (source:Saga {uuid: $source_uuid})
		MATCH (target:Episodic {uuid: $target_uuid})
		MERGE (source)-[e:HAS_EPISODE {uuid: $uuid}]->(target)
		SET e.group_id = $group_id,
			e.created_at = $created_at
		RETURN e.uuid AS uuid
	`

	InvalidateEdgeQuery = `
		MATCH ()-[e:RELATES_TO {uuid: $uuid}]->()
		SET e.invalid_at = $invalid_at
		RETURN e.uuid AS uuid
	`

	GetActiveEdgesQuery = `
		MATCH (source:Entity {uuid: $source_uuid})-[e:RELATES_TO]->(target:Entity {uuid: $target_uuid})
		WHERE e.name = $name AND (e.invalid_at IS NULL OR e.invalid_at = "")
		RETURN e.uuid AS uuid, e.fact AS fact
	`

	GetActiveEdgesFromSourceQuery = `
		MATCH (source:Entity {uuid: $source_uuid})-[e:RELATES_TO]->(target:Entity)
		WHERE (e.invalid_at IS NULL OR e.invalid_at = "")
		RETURN e.uuid AS uuid, e.fact AS fact, e.name AS name, target.uuid AS target_uuid
	`
	
	GetGroupNodesQuery = `
		MATCH (n:Entity {group_id: $group_id})
		RETURN n.uuid AS uuid, n.name AS name, n.summary AS summary
	`

	GetGroupEdgesQuery = `
		MATCH (n:Entity {group_id: $group_id})-[e:RELATES_TO]->(m:Entity {group_id: $group_id})
		WHERE (e.invalid_at IS NULL OR e.invalid_at = "")
		RETURN e.uuid AS uuid, n.uuid AS source_uuid, m.uuid AS target_uuid, e.fact as fact
	`
	
	SaveCommunityEdgeQuery = `
		MATCH (c:Community {uuid: $source_uuid})
		MATCH (e:Entity {uuid: $target_uuid})
		MERGE (c)-[r:HAS_MEMBER {uuid: $uuid}]->(e)
		SET r.group_id = $group_id,
			r.created_at = $created_at
		RETURN r.uuid AS uuid
	`
	GetRecentEpisodesQuery = `
		MATCH (e:Episodic)
		WHERE e.group_id = $group_id
		RETURN e.uuid AS uuid, e.content AS content, e.created_at AS created_at
		ORDER BY e.created_at DESC
		LIMIT $limit
	`
)
