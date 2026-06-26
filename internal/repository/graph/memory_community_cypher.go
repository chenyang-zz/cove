/**
 * @Time   : 2026/6/24 23:48
 * @Author : chenyangzhao542@gmail.com
 * @File   : memory_community_cypher.go
 **/

package graph

const countCommunitiesCypher = `
MATCH (c:Community {user_id: $user_id})
RETURN count(c) AS cnt
`

const listEntityEmbeddingCypher = `
MATCH (e:Entity {user_id: $user_id})
RETURN {
	id: e.id,
	name_embedding: e.name_embedding,
	community_id: e.community_id
} AS entity_embedding
`

const listNeighborsByEntityIdsCypher = `
Match (e:Entity {user_id: $user_id})
WHERE e.id IN $entity_ids
MATCH (e)-[:RELATION]-(nb:Entity {user_id: $user_id})
RETURN {
  entity_id: e.id,
  id: nb.id,
  community_id: nb.community_id,
  name_embedding: nb.name_embedding
} AS neighbor
`

const upsertCommunitiesCypher = `
UNWIND $community_ids AS community_id
MERGE (c:Community {id: community_id, user_id: $user_id})
ON CREATE SET c.created_at = $created_at,
	c.member_count = 0,
	c.name = community_id,
	c.summary = ''
RETURN c.id AS id
`

const assignEntitiesToCommunitiesCypher = `
UNWIND $rows AS row
MATCH (e:Entity {user_id: $user_id, id: row.entity_id})
MATCH (c:Community {user_id: $user_id, id: row.community_id})
SET e.community_id = row.community_id
MERGE (e)-[r:IN_COMMUNITY]->(c)
WITH e, c, r
MATCH (e)-[other:IN_COMMUNITY]->(otherCommunity:Community)
WHERE otherCommunity <> c
DELETE other
`

const refreshCommunityMemberCountCypher = `
UNWIND $community_ids AS community_id
MATCH (c:Community {usr_id: $user_id, id: community_id})
OPTIONAL MATCH (e:Entity {user_id: $user_id, community_id: community_id})
WITH c, count(e) AS cnt
SET c.member_count = cnt
RETURN cnt
`

const getCommunityMembersCypher = `
UNWIND $community_ids AS community_id
MATCH (e:Entity {user_id: $user_id, community_id: community_id})
WITH community_id, collect({
	 id: e.id,
	 name: e.name,
	 type: e.type,
	 description: e.description,
	 aliases: e.aliases,
	 name_embedding: e.name_embedding
}) AS members
RETURN members
`

const pruneEmptyCommunityCypher = `
MATCH (c:Community {user_id: $user_id})
WHERE c.member_count = 0 OR c.member_count IS NULL
DETACH DELETE c
`

const updateCommunityMetadataCypher = `
UNWIND $rows AS row
MATCH (c:Community {user_id: $user_id, id: row.community_id})
SET c.name = row.name,
	c.summary = row.summary
RETURN c.id AS id
`
