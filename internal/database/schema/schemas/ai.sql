--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 18.3
-- Dumped by pgschema version 1.7.4

SET search_path TO ai, public;


--
-- Name: knowledge_bases; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS knowledge_bases (
    id uuid DEFAULT gen_random_uuid(),
    name text NOT NULL,
    namespace text DEFAULT 'default' NOT NULL,
    description text,
    embedding_model text DEFAULT 'text-embedding-3-small',
    embedding_dimensions integer DEFAULT 1536,
    chunk_size integer DEFAULT 512,
    chunk_overlap integer DEFAULT 50,
    chunk_strategy text DEFAULT 'recursive',
    enabled boolean DEFAULT true,
    document_count integer DEFAULT 0,
    total_chunks integer DEFAULT 0,
    source text DEFAULT 'api' NOT NULL,
    created_by uuid,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    visibility text DEFAULT 'private',
    owner_id uuid,
    quota_max_documents integer DEFAULT 1000 NOT NULL,
    quota_max_chunks integer DEFAULT 50000 NOT NULL,
    quota_max_storage_bytes bigint DEFAULT 1073741824 NOT NULL,
    pipeline_type text DEFAULT 'none' NOT NULL,
    pipeline_config jsonb DEFAULT '{}' NOT NULL,
    transformation_function text,
    entity_extraction_enabled boolean DEFAULT true NOT NULL,
    tenant_id uuid,
    CONSTRAINT knowledge_bases_pkey PRIMARY KEY (id),
    CONSTRAINT unique_knowledge_base_name_namespace UNIQUE (name, namespace),
    CONSTRAINT knowledge_bases_pipeline_type_check CHECK (pipeline_type IN ('none'::text, 'sql'::text, 'edge_function'::text, 'webhook'::text)),
    CONSTRAINT knowledge_bases_quota_max_chunks_check CHECK (quota_max_chunks >= 0),
    CONSTRAINT knowledge_bases_quota_max_documents_check CHECK (quota_max_documents >= 0),
    CONSTRAINT knowledge_bases_quota_max_storage_bytes_check CHECK (quota_max_storage_bytes >= 0),
    CONSTRAINT knowledge_bases_source_check CHECK (source IN ('filesystem'::text, 'api'::text, 'sdk'::text)),
    CONSTRAINT knowledge_bases_visibility_check CHECK (visibility IN ('private'::text, 'shared'::text, 'public'::text))
);


COMMENT ON TABLE knowledge_bases IS 'Knowledge base collections for RAG retrieval';


COMMENT ON COLUMN ai.knowledge_bases.chunk_size IS 'Target number of tokens per chunk';


COMMENT ON COLUMN ai.knowledge_bases.chunk_overlap IS 'Number of overlapping tokens between chunks';


COMMENT ON COLUMN ai.knowledge_bases.chunk_strategy IS 'Chunking strategy: recursive (default), sentence, paragraph, or fixed';


COMMENT ON COLUMN ai.knowledge_bases.visibility IS 'private=owner only, shared=explicit permissions, public=all authenticated users';


COMMENT ON COLUMN ai.knowledge_bases.entity_extraction_enabled IS 'When true (default), documents are processed by the rule-based entity extractor on insert/reprocess. Set to false to skip extraction (no entities, no relationships, no graph-boost signal).';


COMMENT ON COLUMN ai.knowledge_bases.quota_max_documents IS 'Maximum documents allowed in this KB';


COMMENT ON COLUMN ai.knowledge_bases.quota_max_chunks IS 'Maximum chunks allowed in this KB';


COMMENT ON COLUMN ai.knowledge_bases.quota_max_storage_bytes IS 'Maximum storage in bytes allowed in this KB';


COMMENT ON COLUMN ai.knowledge_bases.pipeline_type IS 'Type of transformation pipeline: none, sql, edge_function, or webhook';


COMMENT ON COLUMN ai.knowledge_bases.pipeline_config IS 'Configuration for the pipeline (function name, webhook URL, etc.)';


COMMENT ON COLUMN ai.knowledge_bases.transformation_function IS 'Name of SQL transformation function (for pipeline_type=sql)';

--
-- Name: idx_ai_knowledge_bases_enabled; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_knowledge_bases_enabled ON knowledge_bases (enabled);

--
-- Name: idx_ai_knowledge_bases_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_knowledge_bases_name ON knowledge_bases (name);

--
-- Name: idx_ai_knowledge_bases_namespace; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_knowledge_bases_namespace ON knowledge_bases (namespace);

--
-- Name: idx_ai_knowledge_bases_owner; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_knowledge_bases_owner ON knowledge_bases (owner_id) WHERE (owner_id IS NOT NULL);

--
-- Name: idx_ai_knowledge_bases_owner_quotas; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_knowledge_bases_owner_quotas ON knowledge_bases (owner_id) WHERE (owner_id IS NOT NULL);

--
-- Name: idx_ai_knowledge_bases_visibility; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_knowledge_bases_visibility ON knowledge_bases (visibility) WHERE (visibility <> 'private'::text);

--
-- Name: knowledge_base_permissions; Type: TABLE; Schema: -; Owner: -
-- (Created early because knowledge_bases RLS policies reference it)
--

CREATE TABLE IF NOT EXISTS knowledge_base_permissions (
    id uuid DEFAULT gen_random_uuid(),
    knowledge_base_id uuid NOT NULL,
    user_id uuid NOT NULL,
    permission text NOT NULL,
    granted_by uuid,
    granted_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT knowledge_base_permissions_pkey PRIMARY KEY (id),
    CONSTRAINT unique_kb_user_permission UNIQUE (knowledge_base_id, user_id),
    CONSTRAINT knowledge_base_permissions_knowledge_base_id_fkey FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases (id) ON DELETE CASCADE,
    CONSTRAINT knowledge_base_permissions_permission_check CHECK (permission IN ('viewer'::text, 'editor'::text, 'owner'::text))
);


COMMENT ON TABLE knowledge_base_permissions IS 'Granular permissions for shared knowledge bases';


COMMENT ON COLUMN ai.knowledge_base_permissions.permission IS 'viewer=read only, editor=read+write, owner=full control+manage permissions';

--
-- Name: idx_ai_kb_permissions_kb; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_kb_permissions_kb ON knowledge_base_permissions (knowledge_base_id);

--
-- Name: idx_ai_kb_permissions_user; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_kb_permissions_user ON knowledge_base_permissions (user_id);

--
-- Name: knowledge_bases; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE knowledge_bases ENABLE ROW LEVEL SECURITY;

--
-- Name: ai_kb_admin_all; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_kb_admin_all ON knowledge_bases TO authenticated USING (auth.current_user_role() = 'instance_admin') WITH CHECK (auth.current_user_role() = 'instance_admin');

--
-- Name: ai_kb_manage_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_kb_manage_own ON knowledge_bases TO authenticated USING ((owner_id = auth.current_user_id()) OR (EXISTS ( SELECT 1 FROM knowledge_base_permissions WHERE ((knowledge_base_permissions.knowledge_base_id = knowledge_bases.id) AND (knowledge_base_permissions.user_id = auth.current_user_id()) AND (knowledge_base_permissions.permission = ANY (ARRAY['editor', 'owner'])))))) WITH CHECK ((owner_id = auth.current_user_id()) OR (EXISTS ( SELECT 1 FROM knowledge_base_permissions WHERE ((knowledge_base_permissions.knowledge_base_id = knowledge_bases.id) AND (knowledge_base_permissions.user_id = auth.current_user_id()) AND (knowledge_base_permissions.permission = ANY (ARRAY['editor', 'owner']))))));

--
-- Name: ai_kb_read_public; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_kb_read_public ON knowledge_bases FOR SELECT TO authenticated USING (visibility = 'public');

--
-- Name: ai_kb_read_shared; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_kb_read_shared ON knowledge_bases FOR SELECT TO authenticated USING ((visibility = 'shared') AND (EXISTS ( SELECT 1 FROM knowledge_base_permissions WHERE ((knowledge_base_permissions.knowledge_base_id = knowledge_bases.id) AND (knowledge_base_permissions.user_id = auth.current_user_id())))));

--
-- Name: ai_kb_service_all; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_kb_service_all ON knowledge_bases TO authenticated USING (auth.current_user_role() = 'service_role') WITH CHECK (auth.current_user_role() = 'service_role');

--
-- Name: documents; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS documents (
    id uuid DEFAULT gen_random_uuid(),
    knowledge_base_id uuid NOT NULL,
    title text,
    source_url text,
    source_type text DEFAULT 'manual',
    mime_type text,
    content text NOT NULL,
    content_hash text,
    status text DEFAULT 'pending',
    error_message text,
    chunks_count integer DEFAULT 0,
    metadata jsonb DEFAULT '{}',
    tags text[] DEFAULT ARRAY[]::text[],
    created_by uuid,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    indexed_at timestamptz,
    storage_object_id uuid,
    original_filename text,
    owner_id uuid DEFAULT auth.uid(),
    tenant_id uuid,
    CONSTRAINT documents_pkey PRIMARY KEY (id),
    CONSTRAINT documents_knowledge_base_id_fkey FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases (id) ON DELETE CASCADE,
    CONSTRAINT documents_status_check CHECK (status IN ('pending'::text, 'processing'::text, 'indexed'::text, 'failed'::text))
);


COMMENT ON TABLE documents IS 'Source documents in knowledge bases';


COMMENT ON COLUMN ai.documents.content_hash IS 'SHA-256 hash for detecting duplicate or changed content';


COMMENT ON COLUMN ai.documents.metadata IS 'Custom metadata for filtering during retrieval';


COMMENT ON COLUMN ai.documents.storage_object_id IS 'Reference to the uploaded file in storage.objects';


COMMENT ON COLUMN ai.documents.original_filename IS 'Original filename of the uploaded document';


COMMENT ON COLUMN ai.documents.owner_id IS 'User who owns this document (can see and share it). NULL for system-generated documents created via service role.';

--
-- Name: idx_ai_documents_content_hash; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_documents_content_hash ON documents (content_hash);

--
-- Name: idx_ai_documents_knowledge_base; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_documents_knowledge_base ON documents (knowledge_base_id);

--
-- Name: idx_ai_documents_metadata; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_documents_metadata ON documents USING gin (metadata);

--
-- Name: idx_ai_documents_owner; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_documents_owner ON documents (owner_id);

--
-- Name: idx_ai_documents_source_type; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_documents_source_type ON documents (source_type);

--
-- Name: idx_ai_documents_status; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_documents_status ON documents (status);

--
-- Name: idx_ai_documents_storage_object_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_documents_storage_object_id ON documents (storage_object_id);

--
-- Name: idx_ai_documents_tags; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_documents_tags ON documents USING gin (tags);

--
-- Name: idx_ai_documents_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_documents_user_id ON documents ((metadata ->> 'user_id'::text));

--
-- Name: document_permissions; Type: TABLE; Schema: -; Owner: -
-- (Created early because documents RLS policies reference it)
--

CREATE TABLE IF NOT EXISTS document_permissions (
    id uuid DEFAULT gen_random_uuid(),
    document_id uuid NOT NULL,
    user_id uuid NOT NULL,
    permission text NOT NULL,
    granted_by uuid NOT NULL,
    granted_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT document_permissions_pkey PRIMARY KEY (id),
    CONSTRAINT unique_document_user_permission UNIQUE (document_id, user_id),
    CONSTRAINT document_permissions_document_id_fkey FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT document_permissions_permission_check CHECK (permission IN ('viewer'::text, 'editor'::text))
);


COMMENT ON TABLE document_permissions IS 'Permissions for sharing individual documents with specific users';


COMMENT ON COLUMN ai.document_permissions.permission IS 'viewer: can view, editor: can view and edit';

--
-- Name: idx_ai_document_permissions_document; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_document_permissions_document ON document_permissions (document_id);

--
-- Name: idx_ai_document_permissions_user; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_document_permissions_user ON document_permissions (user_id);

--
-- Name: document_permissions; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE document_permissions ENABLE ROW LEVEL SECURITY;

--
-- Name: ai_doc_perms_instance_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_doc_perms_instance_admin ON document_permissions TO authenticated USING (auth.role() = 'instance_admin');

--
-- Name: ai_doc_perms_owner_manage; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_doc_perms_owner_manage ON document_permissions TO authenticated USING (EXISTS ( SELECT 1 FROM documents d WHERE ((d.id = document_permissions.document_id) AND (d.owner_id = auth.uid()))));

--
-- Name: ai_doc_perms_user_read; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_doc_perms_user_read ON document_permissions FOR SELECT TO authenticated USING (user_id = auth.uid());

--
-- Name: documents; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE documents ENABLE ROW LEVEL SECURITY;

--
-- Name: ai_documents_instance_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_documents_instance_admin ON documents TO authenticated USING (auth.role() = 'instance_admin');

--
-- Name: ai_documents_delete_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_documents_delete_own ON documents FOR DELETE TO authenticated USING (owner_id = auth.uid());

--
-- Name: ai_documents_delete_via_kb; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_documents_delete_via_kb ON documents FOR DELETE TO authenticated USING ((EXISTS ( SELECT 1 FROM knowledge_bases kb WHERE ((kb.id = documents.knowledge_base_id) AND (kb.owner_id = auth.current_user_id())))) OR (EXISTS ( SELECT 1 FROM (knowledge_bases kb JOIN knowledge_base_permissions kbp ON ((kb.id = kbp.knowledge_base_id))) WHERE ((kb.id = documents.knowledge_base_id) AND (kbp.user_id = auth.current_user_id()) AND (kbp.permission = ANY (ARRAY['editor', 'owner']))))));

--
-- Name: ai_documents_insert_via_kb; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_documents_insert_via_kb ON documents FOR INSERT TO authenticated WITH CHECK ((EXISTS ( SELECT 1 FROM knowledge_bases kb WHERE ((kb.id = documents.knowledge_base_id) AND (kb.owner_id = auth.current_user_id())))) OR (EXISTS ( SELECT 1 FROM (knowledge_bases kb JOIN knowledge_base_permissions kbp ON ((kb.id = kbp.knowledge_base_id))) WHERE ((kb.id = documents.knowledge_base_id) AND (kbp.user_id = auth.current_user_id()) AND (kbp.permission = ANY (ARRAY['editor', 'owner']))))));

--
-- Name: ai_documents_read_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_documents_read_own ON documents FOR SELECT TO authenticated USING (owner_id = auth.uid());

--
-- Name: ai_documents_read_public; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_documents_read_public ON documents FOR SELECT TO authenticated USING (EXISTS ( SELECT 1 FROM knowledge_bases kb WHERE ((kb.id = documents.knowledge_base_id) AND (kb.visibility = 'public'))));

--
-- Name: ai_documents_read_shared; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_documents_read_shared ON documents FOR SELECT TO authenticated USING (EXISTS ( SELECT 1 FROM document_permissions dp WHERE ((dp.document_id = documents.id) AND (dp.user_id = auth.uid()))));

--
-- Name: ai_documents_read_via_kb; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_documents_read_via_kb ON documents FOR SELECT TO authenticated USING ((EXISTS ( SELECT 1 FROM knowledge_bases kb WHERE ((kb.id = documents.knowledge_base_id) AND (kb.owner_id = auth.current_user_id())))) OR (EXISTS ( SELECT 1 FROM (knowledge_bases kb JOIN knowledge_base_permissions kbp ON ((kb.id = kbp.knowledge_base_id))) WHERE ((kb.id = documents.knowledge_base_id) AND (kbp.user_id = auth.current_user_id())))) OR (EXISTS ( SELECT 1 FROM knowledge_bases kb WHERE ((kb.id = documents.knowledge_base_id) AND (kb.visibility = 'public')))));

--
-- Name: ai_documents_update_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_documents_update_own ON documents FOR UPDATE TO authenticated USING (owner_id = auth.uid());

--
-- Name: ai_documents_update_shared; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_documents_update_shared ON documents FOR UPDATE TO authenticated USING (EXISTS ( SELECT 1 FROM document_permissions dp WHERE ((dp.document_id = documents.id) AND (dp.user_id = auth.uid()) AND (dp.permission = 'editor'))));

--
-- Name: ai_documents_update_via_kb; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_documents_update_via_kb ON documents FOR UPDATE TO authenticated USING ((EXISTS ( SELECT 1 FROM knowledge_bases kb WHERE ((kb.id = documents.knowledge_base_id) AND (kb.owner_id = auth.current_user_id())))) OR (EXISTS ( SELECT 1 FROM (knowledge_bases kb JOIN knowledge_base_permissions kbp ON ((kb.id = kbp.knowledge_base_id))) WHERE ((kb.id = documents.knowledge_base_id) AND (kbp.user_id = auth.current_user_id()) AND (kbp.permission = ANY (ARRAY['editor', 'owner'])))))) WITH CHECK ((EXISTS ( SELECT 1 FROM knowledge_bases kb WHERE ((kb.id = documents.knowledge_base_id) AND (kb.owner_id = auth.current_user_id())))) OR (EXISTS ( SELECT 1 FROM (knowledge_bases kb JOIN knowledge_base_permissions kbp ON ((kb.id = kbp.knowledge_base_id))) WHERE ((kb.id = documents.knowledge_base_id) AND (kbp.user_id = auth.current_user_id()) AND (kbp.permission = ANY (ARRAY['editor', 'owner']))))));

--
-- Name: ai_documents_user_isolation; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_documents_user_isolation ON documents FOR SELECT TO authenticated USING (((metadata ->> 'user_id') IS NULL) OR ((metadata ->> 'user_id') = (current_setting('request.jwt.claims', true)::json ->> 'sub')));

--
-- Name: chunks; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS chunks (
    id uuid DEFAULT gen_random_uuid(),
    document_id uuid NOT NULL,
    knowledge_base_id uuid NOT NULL,
    content text NOT NULL,
    chunk_index integer NOT NULL,
    start_offset integer,
    end_offset integer,
    token_count integer,
    embedding public.vector(1536),
    metadata jsonb DEFAULT '{}',
    created_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT chunks_pkey PRIMARY KEY (id),
    CONSTRAINT unique_chunk_document_index UNIQUE (document_id, chunk_index),
    CONSTRAINT chunks_document_id_fkey FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT chunks_knowledge_base_id_fkey FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases (id) ON DELETE CASCADE
);


COMMENT ON TABLE chunks IS 'Document chunks with vector embeddings for semantic search';


COMMENT ON COLUMN ai.chunks.chunk_index IS 'Zero-based index of this chunk within the document';


COMMENT ON COLUMN ai.chunks.embedding IS 'Vector embedding from configured embedding model';

--
-- Name: idx_ai_chunks_document; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_chunks_document ON chunks (document_id);

--
-- Name: idx_ai_chunks_embedding_cosine; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_chunks_embedding_cosine ON chunks USING ivfflat (embedding vector_cosine_ops);

--
-- Name: idx_ai_chunks_embedding_l2; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_chunks_embedding_l2 ON chunks USING ivfflat (embedding);

--
-- Name: idx_ai_chunks_knowledge_base; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_chunks_knowledge_base ON chunks (knowledge_base_id);

--
-- Name: idx_ai_chunks_metadata; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_chunks_metadata ON chunks USING gin (metadata);

--
-- Name: chunks; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE chunks ENABLE ROW LEVEL SECURITY;

--
-- Name: ai_chunks_instance_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_chunks_instance_admin ON chunks TO authenticated USING (auth.role() = 'instance_admin');

--
-- Name: ai_chunks_read_own_docs; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_chunks_read_own_docs ON chunks FOR SELECT TO authenticated USING (EXISTS ( SELECT 1 FROM documents d WHERE ((d.id = chunks.document_id) AND (d.owner_id = auth.uid()))));

--
-- Name: ai_chunks_read_shared_docs; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_chunks_read_shared_docs ON chunks FOR SELECT TO authenticated USING (EXISTS ( SELECT 1 FROM (documents d JOIN document_permissions dp ON ((dp.document_id = d.id))) WHERE ((d.id = chunks.document_id) AND (dp.user_id = auth.uid()))));

--
-- Name: ai_chunks_user_isolation; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_chunks_user_isolation ON chunks FOR SELECT TO authenticated USING (EXISTS ( SELECT 1 FROM documents d WHERE ((d.id = chunks.document_id) AND (((d.metadata ->> 'user_id') IS NULL) OR ((d.metadata ->> 'user_id') = (current_setting('request.jwt.claims', true)::json ->> 'sub'))))));

--
-- Name: entities; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS entities (
    id uuid DEFAULT gen_random_uuid(),
    knowledge_base_id uuid NOT NULL,
    entity_type text NOT NULL,
    name text NOT NULL,
    canonical_name text,
    aliases text[] DEFAULT ARRAY[]::text[],
    metadata jsonb DEFAULT '{}',
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    tenant_id uuid,
    CONSTRAINT entities_pkey PRIMARY KEY (id),
    CONSTRAINT entity_unique UNIQUE (knowledge_base_id, entity_type, canonical_name),
    CONSTRAINT entities_knowledge_base_id_fkey FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases (id) ON DELETE CASCADE,
    CONSTRAINT entities_entity_type_check CHECK (entity_type IN ('person'::text, 'organization'::text, 'location'::text, 'concept'::text, 'product'::text, 'event'::text, 'table'::text, 'url'::text, 'api_endpoint'::text, 'datetime'::text, 'code_reference'::text, 'error'::text, 'other'::text))
);

--
-- Name: entities_kb_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entities_kb_idx ON entities (knowledge_base_id);

--
-- Name: entities_name_gin_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entities_name_gin_idx ON entities USING gin (to_tsvector('english'::regconfig, canonical_name));

--
-- Name: entities_name_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entities_name_idx ON entities (canonical_name);

--
-- Name: entities_type_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entities_type_idx ON entities (entity_type);

--
-- Name: entities; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entities ENABLE ROW LEVEL SECURITY;

--
-- Name: document_entities; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS document_entities (
    id uuid DEFAULT gen_random_uuid(),
    document_id uuid NOT NULL,
    entity_id uuid NOT NULL,
    mention_count integer DEFAULT 1,
    first_mention_offset integer,
    salience double precision DEFAULT 0.0,
    context text,
    created_at timestamptz DEFAULT now() NOT NULL,
    tenant_id uuid,
    CONSTRAINT document_entities_pkey PRIMARY KEY (id),
    CONSTRAINT document_entity_unique UNIQUE (document_id, entity_id),
    CONSTRAINT document_entities_document_id_fkey FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT document_entities_entity_id_fkey FOREIGN KEY (entity_id) REFERENCES entities (id) ON DELETE CASCADE
);

--
-- Name: document_entities_doc_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS document_entities_doc_idx ON document_entities (document_id);

--
-- Name: document_entities_entity_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS document_entities_entity_idx ON document_entities (entity_id);

--
-- Name: document_entities_salience_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS document_entities_salience_idx ON document_entities (salience DESC);

--
-- Name: document_entities; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE document_entities ENABLE ROW LEVEL SECURITY;

--
-- Name: entity_relationships; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS entity_relationships (
    id uuid DEFAULT gen_random_uuid(),
    knowledge_base_id uuid NOT NULL,
    source_entity_id uuid NOT NULL,
    target_entity_id uuid NOT NULL,
    relationship_type text NOT NULL,
    direction text DEFAULT 'forward' NOT NULL,
    confidence double precision,
    metadata jsonb DEFAULT '{}',
    created_at timestamptz DEFAULT now() NOT NULL,
    tenant_id uuid,
    CONSTRAINT entity_relationships_pkey PRIMARY KEY (id),
    CONSTRAINT relationship_unique UNIQUE (knowledge_base_id, source_entity_id, target_entity_id, relationship_type),
    CONSTRAINT entity_relationships_knowledge_base_id_fkey FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases (id) ON DELETE CASCADE,
    CONSTRAINT entity_relationships_source_entity_id_fkey FOREIGN KEY (source_entity_id) REFERENCES entities (id) ON DELETE CASCADE,
    CONSTRAINT entity_relationships_target_entity_id_fkey FOREIGN KEY (target_entity_id) REFERENCES entities (id) ON DELETE CASCADE,
    CONSTRAINT entity_relationships_confidence_check CHECK (confidence >= 0.0::double precision AND confidence <= 1.0::double precision),
    CONSTRAINT entity_relationships_direction_check CHECK (direction IN ('forward'::text, 'backward'::text, 'bidirectional'::text)),
    CONSTRAINT entity_relationships_relationship_type_check CHECK (relationship_type IN ('works_at'::text, 'located_in'::text, 'founded_by'::text, 'owns'::text, 'part_of'::text, 'related_to'::text, 'knows'::text, 'customer_of'::text, 'supplier_of'::text, 'invested_in'::text, 'acquired'::text, 'merged_with'::text, 'competitor_of'::text, 'parent_of'::text, 'child_of'::text, 'spouse_of'::text, 'sibling_of'::text, 'foreign_key'::text, 'depends_on'::text, 'other'::text)),
    CONSTRAINT no_self_relationship CHECK (source_entity_id <> target_entity_id)
);

--
-- Name: relationships_kb_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS relationships_kb_idx ON entity_relationships (knowledge_base_id);

--
-- Name: relationships_source_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS relationships_source_idx ON entity_relationships (source_entity_id);

--
-- Name: relationships_target_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS relationships_target_idx ON entity_relationships (target_entity_id);

--
-- Name: relationships_type_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS relationships_type_idx ON entity_relationships (relationship_type);

--
-- Name: entity_relationships; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entity_relationships ENABLE ROW LEVEL SECURITY;

--
-- Name: providers; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS providers (
    id uuid DEFAULT gen_random_uuid(),
    name text NOT NULL,
    display_name text NOT NULL,
    provider_type text NOT NULL,
    is_default boolean DEFAULT false,
    config jsonb DEFAULT '{}' NOT NULL,
    enabled boolean DEFAULT true,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    created_by uuid,
    use_for_embeddings boolean,
    embedding_model text,
    read_only boolean DEFAULT false,
    tenant_id uuid,
    CONSTRAINT providers_pkey PRIMARY KEY (id),
    CONSTRAINT providers_name_key UNIQUE (name),
    CONSTRAINT providers_provider_type_check CHECK (provider_type IN ('openai'::text, 'azure'::text, 'ollama'::text, 'anthropic'::text))
);


COMMENT ON TABLE providers IS 'AI provider configurations (OpenAI, Azure, Ollama)';


COMMENT ON COLUMN ai.providers.config IS 'Provider-specific config (api_key, endpoint, model) - should be encrypted at application level';


COMMENT ON COLUMN ai.providers.use_for_embeddings IS 'When true, this provider is explicitly used for embedding generation. NULL means follow default provider (auto mode).';


COMMENT ON COLUMN ai.providers.embedding_model IS 'Embedding model to use for this provider. NULL means use provider-specific default (e.g., text-embedding-3-small for OpenAI).';


COMMENT ON COLUMN ai.providers.read_only IS 'True if provider is configured via environment/YAML and cannot be modified via API';

--
-- Name: idx_ai_providers_enabled; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_providers_enabled ON providers (enabled);

--
-- Name: idx_ai_providers_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_providers_name ON providers (name);

--
-- Name: idx_ai_providers_single_default; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_providers_single_default ON providers (is_default) WHERE (is_default = true);

--
-- Name: idx_ai_providers_single_embedding; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_providers_single_embedding ON providers (use_for_embeddings) WHERE (use_for_embeddings = true);

--
-- Name: idx_ai_providers_type; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_providers_type ON providers (provider_type);

--
-- Name: providers; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE providers ENABLE ROW LEVEL SECURITY;

--
-- Name: ai_providers_read; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_providers_read ON providers FOR SELECT TO authenticated USING (enabled = true);

--
-- Name: chatbots; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS chatbots (
    id uuid DEFAULT gen_random_uuid(),
    name text NOT NULL,
    namespace text DEFAULT 'default' NOT NULL,
    description text,
    code text NOT NULL,
    original_code text,
    is_bundled boolean DEFAULT false,
    bundle_error text,
    allowed_tables text[] DEFAULT ARRAY[]::text[],
    allowed_operations text[] DEFAULT ARRAY['SELECT'],
    allowed_schemas text[] DEFAULT ARRAY['public'],
    enabled boolean DEFAULT true,
    max_tokens integer DEFAULT 4096,
    temperature numeric(3,2) DEFAULT 0.7,
    provider_id uuid,
    persist_conversations boolean DEFAULT false,
    conversation_ttl_hours integer DEFAULT 24,
    max_conversation_turns integer DEFAULT 50,
    rate_limit_per_minute integer DEFAULT 20,
    daily_request_limit integer DEFAULT 500,
    daily_token_budget integer DEFAULT 100000,
    allow_unauthenticated boolean DEFAULT false,
    is_public boolean DEFAULT true,
    http_allowed_domains text[] DEFAULT ARRAY[]::text[],
    version integer DEFAULT 1,
    source text DEFAULT 'filesystem' NOT NULL,
    created_by uuid,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    intent_rules jsonb,
    required_columns jsonb,
    default_table text,
    response_language text DEFAULT 'auto',
    disable_execution_logs boolean DEFAULT false NOT NULL,
    mcp_tools text[] DEFAULT ARRAY[]::text[],
    use_mcp_schema boolean DEFAULT false,
    require_roles text[] DEFAULT ARRAY[]::text[],
    tenant_id uuid,
    CONSTRAINT chatbots_pkey PRIMARY KEY (id),
    CONSTRAINT unique_chatbot_name_namespace UNIQUE (name, namespace),
    CONSTRAINT chatbots_provider_id_fkey FOREIGN KEY (provider_id) REFERENCES providers (id) ON DELETE SET NULL
);


COMMENT ON TABLE chatbots IS 'AI chatbot definitions with system prompts and tool configurations';


COMMENT ON COLUMN ai.chatbots.allowed_tables IS 'Tables the chatbot can query (from @fluxbase:allowed-tables annotation)';


COMMENT ON COLUMN ai.chatbots.allowed_operations IS 'SQL operations allowed (SELECT, INSERT, UPDATE, DELETE)';


COMMENT ON COLUMN ai.chatbots.rate_limit_per_minute IS 'Max requests per minute per user (from @fluxbase:rate-limit annotation)';


COMMENT ON COLUMN ai.chatbots.http_allowed_domains IS 'Allowed domains for HTTP requests (from @fluxbase:http-allowed-domains annotation)';


COMMENT ON COLUMN ai.chatbots.intent_rules IS 'Intent validation rules: [{keywords:[], requiredTable:"", forbiddenTable:""}]';


COMMENT ON COLUMN ai.chatbots.required_columns IS 'Required columns per table: {"table1":["col1","col2"]}';


COMMENT ON COLUMN ai.chatbots.default_table IS 'Default table for queries (from @fluxbase:default-table)';


COMMENT ON COLUMN ai.chatbots.response_language IS 'Response language setting: "auto" (match user language), ISO code (e.g., "en"), or language name (e.g., "German")';


COMMENT ON COLUMN ai.chatbots.disable_execution_logs IS 'When true, execution logs are not created for this chatbot (from @fluxbase:disable-execution-logs annotation)';


COMMENT ON COLUMN ai.chatbots.mcp_tools IS 'List of MCP tools this chatbot can use (e.g., query_table, insert_record, invoke_function)';


COMMENT ON COLUMN ai.chatbots.use_mcp_schema IS 'If true, fetch schema from MCP resources instead of direct DB introspection';


COMMENT ON COLUMN ai.chatbots.require_roles IS 'Required roles to access this chatbot. User needs ANY of the specified roles.';

--
-- Name: idx_ai_chatbots_enabled; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_chatbots_enabled ON chatbots (enabled);

--
-- Name: idx_ai_chatbots_has_intent_rules; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_chatbots_has_intent_rules ON chatbots ((intent_rules IS NOT NULL));

--
-- Name: idx_ai_chatbots_http_domains; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_chatbots_http_domains ON chatbots USING gin (http_allowed_domains);

--
-- Name: idx_ai_chatbots_mcp_tools; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_chatbots_mcp_tools ON chatbots USING gin (mcp_tools);

--
-- Name: idx_ai_chatbots_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_chatbots_name ON chatbots (name);

--
-- Name: idx_ai_chatbots_namespace; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_chatbots_namespace ON chatbots (namespace);

--
-- Name: idx_ai_chatbots_source; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_chatbots_source ON chatbots (source);

--
-- Name: chatbots; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE chatbots ENABLE ROW LEVEL SECURITY;

--
-- Name: ai_chatbots_read; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_chatbots_read ON chatbots FOR SELECT TO authenticated USING ((enabled = true) AND (is_public = true));

--
-- Name: chatbot_knowledge_bases; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS chatbot_knowledge_bases (
    id uuid DEFAULT gen_random_uuid(),
    chatbot_id uuid NOT NULL,
    knowledge_base_id uuid NOT NULL,
    access_level text DEFAULT 'full' NOT NULL,
    filter_expression jsonb DEFAULT '{}',
    context_weight double precision DEFAULT 1.0 NOT NULL,
    priority integer DEFAULT 100,
    intent_keywords text[] DEFAULT ARRAY[]::text[],
    max_chunks integer,
    similarity_threshold double precision,
    enabled boolean DEFAULT true NOT NULL,
    metadata jsonb DEFAULT '{}',
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    tenant_id uuid,
    CONSTRAINT chatbot_knowledge_bases_pkey PRIMARY KEY (id),
    CONSTRAINT chatbot_kb_unique UNIQUE (chatbot_id, knowledge_base_id),
    CONSTRAINT chatbot_knowledge_bases_chatbot_id_fkey FOREIGN KEY (chatbot_id) REFERENCES chatbots (id) ON DELETE CASCADE,
    CONSTRAINT chatbot_knowledge_bases_knowledge_base_id_fkey FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases (id) ON DELETE CASCADE,
    CONSTRAINT chatbot_knowledge_bases_access_level_check CHECK (access_level IN ('full'::text, 'filtered'::text, 'tiered'::text)),
    CONSTRAINT chatbot_knowledge_bases_context_weight_check CHECK (context_weight >= 0.0::double precision AND context_weight <= 1.0::double precision),
    CONSTRAINT chatbot_knowledge_bases_max_chunks_check CHECK (max_chunks > 0 OR max_chunks IS NULL),
    CONSTRAINT chatbot_knowledge_bases_priority_check CHECK (priority >= 1 AND priority <= 1000),
    CONSTRAINT chatbot_knowledge_bases_similarity_threshold_check CHECK (similarity_threshold >= 0.0::double precision AND similarity_threshold <= 1.0::double precision OR similarity_threshold IS NULL)
);

--
-- Name: chatbot_kb_links_chatbot_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS chatbot_kb_links_chatbot_idx ON chatbot_knowledge_bases (chatbot_id) WHERE (enabled = true);

--
-- Name: chatbot_kb_links_kb_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS chatbot_kb_links_kb_idx ON chatbot_knowledge_bases (knowledge_base_id) WHERE (enabled = true);

--
-- Name: chatbot_kb_links_priority_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS chatbot_kb_links_priority_idx ON chatbot_knowledge_bases (chatbot_id, priority) WHERE (enabled = true) AND (access_level = 'tiered'::text);

--
-- Name: chatbot_knowledge_bases; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE chatbot_knowledge_bases ENABLE ROW LEVEL SECURITY;

--
-- Name: conversations; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS conversations (
    id uuid DEFAULT gen_random_uuid(),
    chatbot_id uuid NOT NULL,
    user_id uuid,
    session_id text,
    title text,
    status text DEFAULT 'active',
    turn_count integer DEFAULT 0,
    total_prompt_tokens integer DEFAULT 0,
    total_completion_tokens integer DEFAULT 0,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    last_message_at timestamptz DEFAULT now(),
    expires_at timestamptz,
    tenant_id uuid,
    CONSTRAINT conversations_pkey PRIMARY KEY (id),
    CONSTRAINT conversations_chatbot_id_fkey FOREIGN KEY (chatbot_id) REFERENCES chatbots (id) ON DELETE CASCADE,
    CONSTRAINT conversations_status_check CHECK (status IN ('active'::text, 'archived'::text, 'deleted'::text))
);


COMMENT ON TABLE conversations IS 'AI conversation sessions with token tracking';

--
-- Name: idx_ai_conversations_chatbot; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_conversations_chatbot ON conversations (chatbot_id);

--
-- Name: idx_ai_conversations_expires; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_conversations_expires ON conversations (expires_at) WHERE (expires_at IS NOT NULL);

--
-- Name: idx_ai_conversations_session; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_conversations_session ON conversations (session_id) WHERE (session_id IS NOT NULL);

--
-- Name: idx_ai_conversations_status; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_conversations_status ON conversations (status);

--
-- Name: idx_ai_conversations_user; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_conversations_user ON conversations (user_id);

--
-- Name: conversations; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE conversations ENABLE ROW LEVEL SECURITY;

--
-- Name: ai_conversations_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_conversations_own ON conversations TO authenticated USING (user_id = auth.current_user_id()) WITH CHECK (user_id = auth.current_user_id());

--
-- Name: messages; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS messages (
    id uuid DEFAULT gen_random_uuid(),
    conversation_id uuid NOT NULL,
    role text NOT NULL,
    content text NOT NULL,
    tool_call_id text,
    tool_name text,
    tool_input jsonb,
    tool_output jsonb,
    executed_sql text,
    sql_result_summary text,
    sql_row_count integer,
    sql_error text,
    sql_duration_ms integer,
    prompt_tokens integer,
    completion_tokens integer,
    query_results jsonb,
    metadata jsonb,
    created_at timestamptz DEFAULT now(),
    sequence_number integer NOT NULL,
    tenant_id uuid,
    CONSTRAINT messages_pkey PRIMARY KEY (id),
    CONSTRAINT messages_conversation_id_fkey FOREIGN KEY (conversation_id) REFERENCES conversations (id) ON DELETE CASCADE,
    CONSTRAINT messages_role_check CHECK (role IN ('user'::text, 'assistant'::text, 'system'::text, 'tool'::text))
);


COMMENT ON TABLE messages IS 'Individual messages within AI conversations';


COMMENT ON COLUMN ai.messages.query_results IS 'Array of query results with query, summary, row_count, and data for assistant messages';

--
-- Name: idx_ai_messages_conversation; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_messages_conversation ON messages (conversation_id);

--
-- Name: idx_ai_messages_has_query_results; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_messages_has_query_results ON messages (conversation_id) WHERE (query_results IS NOT NULL);

--
-- Name: idx_ai_messages_role; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_messages_role ON messages (role);

--
-- Name: idx_ai_messages_sequence; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_messages_sequence ON messages (conversation_id, sequence_number);

--
-- Name: messages; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE messages ENABLE ROW LEVEL SECURITY;

--
-- Name: ai_messages_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_messages_own ON messages TO authenticated USING (conversation_id IN ( SELECT conversations.id FROM conversations WHERE (conversations.user_id = auth.current_user_id()))) WITH CHECK (conversation_id IN ( SELECT conversations.id FROM conversations WHERE (conversations.user_id = auth.current_user_id())));

--
-- Name: query_audit_log; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS query_audit_log (
    id uuid DEFAULT gen_random_uuid(),
    chatbot_id uuid,
    conversation_id uuid,
    message_id uuid,
    user_id uuid,
    generated_sql text NOT NULL,
    sanitized_sql text,
    executed boolean DEFAULT false,
    validation_passed boolean,
    validation_errors text[],
    success boolean,
    error_message text,
    rows_returned integer,
    execution_duration_ms integer,
    tables_accessed text[],
    operations_used text[],
    rls_user_id text,
    rls_role text,
    ip_address inet,
    user_agent text,
    created_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT query_audit_log_pkey PRIMARY KEY (id),
    CONSTRAINT query_audit_log_chatbot_id_fkey FOREIGN KEY (chatbot_id) REFERENCES chatbots (id) ON DELETE SET NULL,
    CONSTRAINT query_audit_log_conversation_id_fkey FOREIGN KEY (conversation_id) REFERENCES conversations (id) ON DELETE SET NULL,
    CONSTRAINT query_audit_log_message_id_fkey FOREIGN KEY (message_id) REFERENCES messages (id) ON DELETE SET NULL
);


COMMENT ON TABLE query_audit_log IS 'Audit log for all SQL queries generated and executed by AI chatbots';

--
-- Name: idx_ai_query_audit_chatbot; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_query_audit_chatbot ON query_audit_log (chatbot_id);

--
-- Name: idx_ai_query_audit_created; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_query_audit_created ON query_audit_log (created_at DESC);

--
-- Name: idx_ai_query_audit_executed; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_query_audit_executed ON query_audit_log (executed);

--
-- Name: idx_ai_query_audit_success; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_query_audit_success ON query_audit_log (success);

--
-- Name: idx_ai_query_audit_user; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_query_audit_user ON query_audit_log (user_id);

--
-- Name: tool_audit_log; Type: TABLE; Schema: -; Owner: -
--
-- Audit log for non-SQL tool calls (web_search, fetch_url, invoke_rpc,
-- invoke_function, custom tools). Mirrors query_audit_log but records the tool
-- name, arguments, and result summary so every agent action is inspectable
-- after the fact (the SQL-only audit log misses web/RPC/function calls, which
-- made Tavily-not-firing failures invisible).
--

CREATE TABLE IF NOT EXISTS tool_audit_log (
    id uuid DEFAULT gen_random_uuid(),
    chatbot_id uuid,
    conversation_id uuid,
    message_id uuid,
    user_id uuid,
    tool_name text NOT NULL,
    tool_type text NOT NULL,
    arguments jsonb,
    success boolean,
    error_message text,
    result_summary text,
    result_meta jsonb,
    duration_ms integer,
    agent text,
    created_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT tool_audit_log_pkey PRIMARY KEY (id),
    CONSTRAINT tool_audit_log_chatbot_id_fkey FOREIGN KEY (chatbot_id) REFERENCES chatbots (id) ON DELETE SET NULL,
    CONSTRAINT tool_audit_log_conversation_id_fkey FOREIGN KEY (conversation_id) REFERENCES conversations (id) ON DELETE SET NULL,
    CONSTRAINT tool_audit_log_message_id_fkey FOREIGN KEY (message_id) REFERENCES messages (id) ON DELETE SET NULL
);

COMMENT ON TABLE tool_audit_log IS 'Audit log for non-SQL AI tool calls (web_search, fetch_url, invoke_rpc, invoke_function, custom)';

CREATE INDEX IF NOT EXISTS idx_ai_tool_audit_chatbot ON tool_audit_log (chatbot_id);
CREATE INDEX IF NOT EXISTS idx_ai_tool_audit_created ON tool_audit_log (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_tool_audit_conversation ON tool_audit_log (conversation_id);
CREATE INDEX IF NOT EXISTS idx_ai_tool_audit_tool ON tool_audit_log (tool_name);
CREATE INDEX IF NOT EXISTS idx_ai_tool_audit_user ON tool_audit_log (user_id);

--
-- Name: query_audit_log; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE query_audit_log ENABLE ROW LEVEL SECURITY;

--
-- Name: retrieval_log; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS retrieval_log (
    id uuid DEFAULT gen_random_uuid(),
    chatbot_id uuid,
    conversation_id uuid,
    knowledge_base_id uuid,
    user_id uuid,
    query_text text NOT NULL,
    query_embedding_model text,
    chunks_retrieved integer DEFAULT 0,
    chunk_ids uuid[],
    similarity_scores double precision[],
    retrieval_duration_ms integer,
    created_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT retrieval_log_pkey PRIMARY KEY (id),
    CONSTRAINT retrieval_log_chatbot_id_fkey FOREIGN KEY (chatbot_id) REFERENCES chatbots (id) ON DELETE SET NULL,
    CONSTRAINT retrieval_log_conversation_id_fkey FOREIGN KEY (conversation_id) REFERENCES conversations (id) ON DELETE SET NULL,
    CONSTRAINT retrieval_log_knowledge_base_id_fkey FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases (id) ON DELETE SET NULL
);


COMMENT ON TABLE retrieval_log IS 'Audit log for RAG retrieval operations';

--
-- Name: idx_ai_retrieval_log_chatbot; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_retrieval_log_chatbot ON retrieval_log (chatbot_id);

--
-- Name: idx_ai_retrieval_log_created; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_retrieval_log_created ON retrieval_log (created_at DESC);

--
-- Name: idx_ai_retrieval_log_kb; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_retrieval_log_kb ON retrieval_log (knowledge_base_id);

--
-- Name: retrieval_log; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE retrieval_log ENABLE ROW LEVEL SECURITY;

--
-- Name: ai_retrieval_log_instance_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_retrieval_log_instance_admin ON retrieval_log FOR SELECT TO authenticated USING (auth.role() = 'instance_admin');

--
-- Name: table_export_sync_configs; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS table_export_sync_configs (
    id uuid DEFAULT gen_random_uuid(),
    knowledge_base_id uuid NOT NULL,
    schema_name text NOT NULL,
    table_name text NOT NULL,
    columns text[],
    sync_mode text DEFAULT 'manual' NOT NULL,
    sync_on_insert boolean DEFAULT true,
    sync_on_update boolean DEFAULT true,
    sync_on_delete boolean DEFAULT false,
    debounce_seconds integer DEFAULT 60,
    include_foreign_keys boolean DEFAULT true,
    include_indexes boolean DEFAULT false,
    last_sync_at timestamptz,
    last_sync_status text,
    last_sync_error text,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT table_export_sync_configs_pkey PRIMARY KEY (id),
    CONSTRAINT table_export_sync_configs_knowledge_base_id_schema_name_tab_key UNIQUE (knowledge_base_id, schema_name, table_name),
    CONSTRAINT table_export_sync_configs_knowledge_base_id_fkey FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases (id) ON DELETE CASCADE,
    CONSTRAINT table_export_sync_configs_sync_mode_check CHECK (sync_mode IN ('manual'::text, 'automatic'::text))
);

--
-- Name: idx_table_export_sync_kb; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_table_export_sync_kb ON table_export_sync_configs (knowledge_base_id);

--
-- Name: idx_table_export_sync_mode; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_table_export_sync_mode ON table_export_sync_configs (sync_mode) WHERE (sync_mode = 'automatic'::text);

--
-- Name: idx_table_export_sync_table; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_table_export_sync_table ON table_export_sync_configs (schema_name, table_name);

--
-- Name: table_export_sync_configs; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE table_export_sync_configs ENABLE ROW LEVEL SECURITY;

--
-- Name: Users can delete sync configs for their knowledge bases; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY "Users can delete sync configs for their knowledge bases" ON table_export_sync_configs FOR DELETE TO PUBLIC USING (knowledge_base_id IN ( SELECT knowledge_bases.id FROM knowledge_bases WHERE ((knowledge_bases.owner_id = auth.uid()) OR (knowledge_bases.owner_id IS NULL) OR (EXISTS ( SELECT 1 FROM (documents d JOIN document_permissions dp ON ((dp.document_id = d.id))) WHERE ((d.knowledge_base_id = table_export_sync_configs.knowledge_base_id) AND (dp.user_id = auth.uid()) AND (dp.permission = 'editor')))))));

--
-- Name: Users can insert sync configs for their knowledge bases; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY "Users can insert sync configs for their knowledge bases" ON table_export_sync_configs FOR INSERT TO PUBLIC WITH CHECK (knowledge_base_id IN ( SELECT knowledge_bases.id FROM knowledge_bases WHERE ((knowledge_bases.owner_id = auth.uid()) OR (knowledge_bases.owner_id IS NULL) OR (EXISTS ( SELECT 1 FROM (documents d JOIN document_permissions dp ON ((dp.document_id = d.id))) WHERE ((d.knowledge_base_id = table_export_sync_configs.knowledge_base_id) AND (dp.user_id = auth.uid()) AND (dp.permission = 'editor')))))));

--
-- Name: Users can update sync configs for their knowledge bases; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY "Users can update sync configs for their knowledge bases" ON table_export_sync_configs FOR UPDATE TO PUBLIC USING (knowledge_base_id IN ( SELECT knowledge_bases.id FROM knowledge_bases WHERE ((knowledge_bases.owner_id = auth.uid()) OR (knowledge_bases.owner_id IS NULL) OR (EXISTS ( SELECT 1 FROM (documents d JOIN document_permissions dp ON ((dp.document_id = d.id))) WHERE ((d.knowledge_base_id = table_export_sync_configs.knowledge_base_id) AND (dp.user_id = auth.uid()) AND (dp.permission = 'editor')))))));

--
-- Name: Users can view sync configs for their knowledge bases; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY "Users can view sync configs for their knowledge bases" ON table_export_sync_configs FOR SELECT TO PUBLIC USING (knowledge_base_id IN ( SELECT knowledge_bases.id FROM knowledge_bases WHERE ((knowledge_bases.owner_id = auth.uid()) OR (knowledge_bases.owner_id IS NULL) OR (knowledge_bases.visibility = 'public') OR (EXISTS ( SELECT 1 FROM (documents d JOIN document_permissions dp ON ((dp.document_id = d.id))) WHERE ((d.knowledge_base_id = table_export_sync_configs.knowledge_base_id) AND (dp.user_id = auth.uid())))))));

--
-- Name: user_chatbot_usage; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS user_chatbot_usage (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    chatbot_id uuid NOT NULL,
    date date DEFAULT CURRENT_DATE NOT NULL,
    request_count integer DEFAULT 0,
    tokens_used integer DEFAULT 0,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT user_chatbot_usage_pkey PRIMARY KEY (id),
    CONSTRAINT user_chatbot_usage_user_id_chatbot_id_date_key UNIQUE (user_id, chatbot_id, date),
    CONSTRAINT user_chatbot_usage_chatbot_id_fkey FOREIGN KEY (chatbot_id) REFERENCES chatbots (id) ON DELETE CASCADE
);


COMMENT ON TABLE user_chatbot_usage IS 'Daily usage tracking per user per chatbot for rate limiting';

--
-- Name: idx_ai_usage_date; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_usage_date ON user_chatbot_usage (date);

--
-- Name: idx_ai_usage_lookup; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_usage_lookup ON user_chatbot_usage (user_id, chatbot_id, date);

--
-- Name: user_chatbot_usage; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE user_chatbot_usage ENABLE ROW LEVEL SECURITY;

--
-- Name: ai_usage_own_read; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_usage_own_read ON user_chatbot_usage FOR SELECT TO authenticated USING (user_id = auth.current_user_id());

--
-- Name: user_provider_preferences; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS user_provider_preferences (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    provider_id uuid,
    api_key_encrypted text,
    model_override text,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT user_provider_preferences_pkey PRIMARY KEY (id),
    CONSTRAINT user_provider_preferences_user_id_key UNIQUE (user_id),
    CONSTRAINT user_provider_preferences_provider_id_fkey FOREIGN KEY (provider_id) REFERENCES providers (id) ON DELETE SET NULL
);


COMMENT ON TABLE user_provider_preferences IS 'User-level AI provider overrides (when enabled)';

--
-- Name: idx_ai_user_prefs_user; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_user_prefs_user ON user_provider_preferences (user_id);

--
-- Name: user_provider_preferences; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE user_provider_preferences ENABLE ROW LEVEL SECURITY;

--
-- Name: ai_user_prefs_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY ai_user_prefs_own ON user_provider_preferences TO authenticated USING (user_id = auth.current_user_id()) WITH CHECK (user_id = auth.current_user_id());

--
-- Name: user_quotas; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS user_quotas (
    user_id uuid,
    max_documents integer DEFAULT 10000 NOT NULL,
    max_chunks integer DEFAULT 500000 NOT NULL,
    max_storage_bytes bigint DEFAULT 10737418240 NOT NULL,
    used_documents integer DEFAULT 0 NOT NULL,
    used_chunks integer DEFAULT 0 NOT NULL,
    used_storage_bytes bigint DEFAULT 0 NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    tenant_id uuid,
    CONSTRAINT user_quotas_pkey PRIMARY KEY (user_id),
    CONSTRAINT user_quotas_max_chunks_check CHECK (max_chunks >= 0),
    CONSTRAINT user_quotas_max_documents_check CHECK (max_documents >= 0),
    CONSTRAINT user_quotas_max_storage_bytes_check CHECK (max_storage_bytes >= 0),
    CONSTRAINT user_quotas_used_chunks_check CHECK (used_chunks >= 0),
    CONSTRAINT user_quotas_used_documents_check CHECK (used_documents >= 0),
    CONSTRAINT user_quotas_used_storage_bytes_check CHECK (used_storage_bytes >= 0)
);


COMMENT ON TABLE user_quotas IS 'Per-user resource quotas for knowledge bases';


COMMENT ON COLUMN ai.user_quotas.max_documents IS 'Maximum number of documents allowed across all user KBs';


COMMENT ON COLUMN ai.user_quotas.max_chunks IS 'Maximum number of chunks allowed across all user KBs';


COMMENT ON COLUMN ai.user_quotas.max_storage_bytes IS 'Maximum storage in bytes allowed across all user KBs';


COMMENT ON COLUMN ai.user_quotas.used_documents IS 'Current document count across all user KBs';


COMMENT ON COLUMN ai.user_quotas.used_chunks IS 'Current chunk count across all user KBs';


COMMENT ON COLUMN ai.user_quotas.used_storage_bytes IS 'Current storage in bytes across all user KBs';

--
-- Name: idx_ai_user_quotas_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_ai_user_quotas_user_id ON user_quotas (user_id);

--
-- Name: can_access_document(uuid, uuid); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION can_access_document(
    p_document_id uuid,
    p_user_id uuid
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
AS $$
BEGIN
    -- User owns the document
    IF EXISTS (
        SELECT 1 FROM documents
        WHERE id = p_document_id
        AND owner_id = p_user_id
    ) THEN
        RETURN true;
    END IF;

    -- Document is shared with user
    IF EXISTS (
        SELECT 1 FROM document_permissions
        WHERE document_id = p_document_id
        AND user_id = p_user_id
    ) THEN
        RETURN true;
    END IF;

    RETURN false;
END;
$$;

--
-- Name: can_access_document(uuid, uuid); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION can_access_document(uuid, uuid) IS 'Check if a user can access a document (owns it or has been granted permission)';

--
-- Name: find_related_entities(uuid, uuid, integer, text[]); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION find_related_entities(
    p_kb_id uuid,
    p_entity_id uuid,
    p_max_depth integer DEFAULT 2,
    p_relationship_types text[] DEFAULT NULL
)
RETURNS TABLE(entity_id uuid, entity_type text, name text, canonical_name text, relationship_type text, depth integer, path text[])
LANGUAGE plpgsql
VOLATILE
AS $$
DECLARE
    v_max_depth INTEGER := GREATEST(LEAST(p_max_depth, 5), 1); -- Limit to depth 5
BEGIN
    RETURN QUERY
    WITH RECURSIVE graph_traversal AS (
        -- Base case: direct relationships
        SELECT
            e.id,
            e.entity_type,
            e.name,
            e.canonical_name,
            r.relationship_type,
            1::INTEGER as depth,
            ARRAY[p_entity_id, e.id]::UUID[] as path
        FROM entity_relationships r
        JOIN entities e ON e.id = r.target_entity_id
        WHERE r.source_entity_id = p_entity_id
            AND r.knowledge_base_id = p_kb_id
            AND (p_relationship_types IS NULL OR r.relationship_type = ANY(p_relationship_types))

        UNION ALL

        -- Recursive case: traverse to depth N
        SELECT
            e.id,
            e.entity_type,
            e.name,
            e.canonical_name,
            r.relationship_type,
            gt.depth + 1,
            gt.path || e.id
        FROM entity_relationships r
        JOIN entities e ON e.id = r.target_entity_id
        JOIN graph_traversal gt ON gt.entity_id = r.source_entity_id
        WHERE r.knowledge_base_id = p_kb_id
            AND gt.depth < v_max_depth
            AND (p_relationship_types IS NULL OR r.relationship_type = ANY(p_relationship_types))
            AND NOT (e.id = ANY(gt.path)) -- Prevent cycles
    )
    SELECT * FROM graph_traversal
    ORDER BY depth, relationship_type, name;
END;
$$;

--
-- Name: generate_span_id(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION generate_span_id()
RETURNS text
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    RETURN 'span_' || encode(gen_random_bytes(8), 'hex');
END;
$$;

--
-- Name: generate_trace_id(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION generate_trace_id()
RETURNS text
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    RETURN 'trace_' || encode(gen_random_bytes(16), 'hex');
END;
$$;

--
-- Name: search_chatbot_knowledge(uuid, vector); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION search_chatbot_knowledge(
    p_chatbot_id uuid,
    p_query_embedding vector
)
RETURNS TABLE(chunk_id uuid, document_id uuid, knowledge_base_id uuid, knowledge_base_name text, content text, similarity double precision, metadata jsonb)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id as chunk_id,
        c.document_id,
        c.knowledge_base_id,
        kb.name as knowledge_base_name,
        c.content,
        1 - (c.embedding <=> p_query_embedding) as similarity,
        c.metadata
    FROM chatbot_knowledge_bases ckb
    JOIN knowledge_bases kb ON kb.id = ckb.knowledge_base_id
    JOIN chunks c ON c.knowledge_base_id = kb.id
    WHERE ckb.chatbot_id = p_chatbot_id
      AND ckb.enabled = true
      AND kb.enabled = true
      AND 1 - (c.embedding <=> p_query_embedding) >= ckb.similarity_threshold
    ORDER BY ckb.priority DESC, c.embedding <=> p_query_embedding
    LIMIT (
        SELECT SUM(max_chunks) FROM chatbot_knowledge_bases
        WHERE chatbot_id = p_chatbot_id AND enabled = true
    );
END;
$$;

--
-- Name: search_chatbot_knowledge(uuid, vector); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION search_chatbot_knowledge(uuid, vector) IS 'Search all knowledge bases linked to a chatbot';

--
-- Name: search_chunks(uuid, vector, integer, double precision); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION search_chunks(
    p_knowledge_base_id uuid,
    p_query_embedding vector,
    p_limit integer DEFAULT 5,
    p_threshold double precision DEFAULT 0.7
)
RETURNS TABLE(chunk_id uuid, document_id uuid, content text, similarity double precision, metadata jsonb)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id as chunk_id,
        c.document_id,
        c.content,
        1 - (c.embedding <=> p_query_embedding) as similarity,
        c.metadata
    FROM chunks c
    WHERE c.knowledge_base_id = p_knowledge_base_id
      AND 1 - (c.embedding <=> p_query_embedding) >= p_threshold
    ORDER BY c.embedding <=> p_query_embedding
    LIMIT p_limit;
END;
$$;

--
-- Name: search_chunks(uuid, vector, integer, double precision); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION search_chunks(uuid, vector, integer, double precision) IS 'Search chunks in a knowledge base by vector similarity';

--
-- Name: search_entities(uuid, text, text[], integer); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION search_entities(
    p_kb_id uuid,
    p_query text,
    p_entity_types text[] DEFAULT NULL,
    p_limit integer DEFAULT 20
)
RETURNS TABLE(entity_id uuid, entity_type text, name text, canonical_name text, aliases text[], rank real)
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    RETURN QUERY
    SELECT
        e.id,
        e.entity_type,
        e.name,
        e.canonical_name,
        e.aliases,
        CASE
            WHEN e.canonical_name ILIKE p_query || '%' THEN 1.0
            WHEN e.name ILIKE p_query || '%' THEN 0.9
            WHEN e.canonical_name ILIKE '%' || p_query || '%' THEN 0.7
            WHEN EXISTS (SELECT 1 FROM unnest(e.aliases) alias WHERE alias ILIKE '%' || p_query || '%') THEN 0.6
            ELSE ts_rank(to_tsvector('english', e.canonical_name), plainto_tsquery('english', p_query)) * 0.5
        END::REAL as rank
    FROM entities e
    WHERE e.knowledge_base_id = p_kb_id
        AND (p_entity_types IS NULL OR e.entity_type = ANY(p_entity_types))
        AND (
            e.canonical_name ILIKE '%' || p_query || '%'
            OR e.name ILIKE '%' || p_query || '%'
            OR EXISTS (SELECT 1 FROM unnest(e.aliases) alias WHERE alias ILIKE '%' || p_query || '%')
            OR to_tsvector('english', e.canonical_name) @@ plainto_tsquery('english', p_query)
        )
    ORDER BY rank DESC
    LIMIT p_limit;
END;
$$;

--
-- Name: set_chunk_user_id(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION set_chunk_user_id()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
AS $$
DECLARE
    v_user_id TEXT;
    v_role TEXT;
BEGIN
    -- Get current user ID from JWT claims
    BEGIN
        v_user_id := current_setting('request.jwt.claims', true)::json->>'sub';
        v_role := current_setting('request.jwt.claims', true)::json->>'role';
    EXCEPTION WHEN OTHERS THEN
        v_user_id := NULL;
        v_role := NULL;
    END;

    -- Skip if service_role
    IF v_role = 'service_role' THEN
        RETURN NEW;
    END IF;

    -- Initialize metadata if NULL
    IF NEW.metadata IS NULL THEN
        NEW.metadata = '{}'::jsonb;
    END IF;

    -- Only set user_id if not already present AND we have a user context
    IF NEW.metadata->>'user_id' IS NULL AND v_user_id IS NOT NULL THEN
        NEW.metadata = jsonb_set(
            NEW.metadata,
            '{user_id}',
            to_jsonb(v_user_id)
        );
    END IF;

    RETURN NEW;
END;
$$;

--
-- Name: set_chunk_user_id(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION set_chunk_user_id() IS 'Auto-populates user_id in chunk metadata from JWT claims for RLS enforcement';

--
-- Name: set_document_owner(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION set_document_owner()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
AS $$
BEGIN
    IF NEW.owner_id IS NULL AND auth.uid() IS NOT NULL THEN
        NEW.owner_id = auth.uid();
    END IF;
    RETURN NEW;
END;
$$;

--
-- Name: set_document_user_id(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION set_document_user_id()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
AS $$
DECLARE
    v_user_id TEXT;
    v_role TEXT;
BEGIN
    -- Get current user ID from JWT claims
    BEGIN
        v_user_id := current_setting('request.jwt.claims', true)::json->>'sub';
        v_role := current_setting('request.jwt.claims', true)::json->>'role';
    EXCEPTION WHEN OTHERS THEN
        v_user_id := NULL;
        v_role := NULL;
    END;

    -- Skip if service_role (admin operations may want global documents)
    -- Or if user_id is already explicitly set in metadata
    IF v_role = 'service_role' THEN
        -- Service role can create global documents
        RETURN NEW;
    END IF;

    -- Initialize metadata if NULL
    IF NEW.metadata IS NULL THEN
        NEW.metadata = '{}'::jsonb;
    END IF;

    -- Only set user_id if not already present AND we have a user context
    IF NEW.metadata->>'user_id' IS NULL AND v_user_id IS NOT NULL THEN
        NEW.metadata = jsonb_set(
            NEW.metadata,
            '{user_id}',
            to_jsonb(v_user_id)
        );
    END IF;

    RETURN NEW;
END;
$$;

--
-- Name: set_document_user_id(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION set_document_user_id() IS 'Auto-populates user_id in document metadata from JWT claims for RLS enforcement';

--
-- Name: update_chatbot_kb_link_updated_at(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_chatbot_kb_link_updated_at()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

--
-- Name: update_entities_updated_at(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_entities_updated_at()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

--
-- Name: update_knowledge_base_chunk_count(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_knowledge_base_chunk_count()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE knowledge_bases
        SET total_chunks = total_chunks + 1
        WHERE id = NEW.knowledge_base_id;
        UPDATE documents
        SET chunks_count = chunks_count + 1
        WHERE id = NEW.document_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE knowledge_bases
        SET total_chunks = GREATEST(0, total_chunks - 1)
        WHERE id = OLD.knowledge_base_id;
        UPDATE documents
        SET chunks_count = GREATEST(0, chunks_count - 1)
        WHERE id = OLD.document_id;
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$;

--
-- Name: update_knowledge_base_counts(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_knowledge_base_counts()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE knowledge_bases
        SET document_count = document_count + 1,
            updated_at = NOW()
        WHERE id = NEW.knowledge_base_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE knowledge_bases
        SET document_count = GREATEST(0, document_count - 1),
            updated_at = NOW()
        WHERE id = OLD.knowledge_base_id;
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$;

--
-- Name: update_knowledge_base_updated_at(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_knowledge_base_updated_at()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

--
-- Name: update_table_export_sync_updated_at(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_table_export_sync_updated_at()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

--
-- Multi-tenancy: Add tenant_id column, FORCE ROW LEVEL SECURITY, tenant policy,
-- and auto-populate trigger for all ai tables
--

-- knowledge_bases
CREATE INDEX IF NOT EXISTS idx_ai_knowledge_bases_tenant_id ON knowledge_bases (tenant_id);

ALTER TABLE knowledge_bases FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_knowledge_bases_tenant ON knowledge_bases TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_knowledge_bases_set_tenant_id
    BEFORE INSERT ON knowledge_bases
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- knowledge_base_permissions
CREATE INDEX IF NOT EXISTS idx_ai_knowledge_base_permissions_tenant_id ON knowledge_base_permissions (tenant_id);

ALTER TABLE knowledge_base_permissions FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_knowledge_base_permissions_tenant ON knowledge_base_permissions TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_knowledge_base_permissions_set_tenant_id
    BEFORE INSERT ON knowledge_base_permissions
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- documents
CREATE INDEX IF NOT EXISTS idx_ai_documents_tenant_id ON documents (tenant_id);

ALTER TABLE documents FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_documents_tenant ON documents TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_documents_set_tenant_id
    BEFORE INSERT ON documents
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- document_permissions
CREATE INDEX IF NOT EXISTS idx_ai_document_permissions_tenant_id ON document_permissions (tenant_id);

ALTER TABLE document_permissions FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_document_permissions_tenant ON document_permissions TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_document_permissions_set_tenant_id
    BEFORE INSERT ON document_permissions
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- chunks
CREATE INDEX IF NOT EXISTS idx_ai_chunks_tenant_id ON chunks (tenant_id);

ALTER TABLE chunks FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_chunks_tenant ON chunks TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_chunks_set_tenant_id
    BEFORE INSERT ON chunks
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- entities
CREATE INDEX IF NOT EXISTS idx_ai_entities_tenant_id ON entities (tenant_id);

ALTER TABLE entities FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_entities_tenant ON entities TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_entities_set_tenant_id
    BEFORE INSERT ON entities
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- document_entities
CREATE INDEX IF NOT EXISTS idx_ai_document_entities_tenant_id ON document_entities (tenant_id);

ALTER TABLE document_entities FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_document_entities_tenant ON document_entities TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_document_entities_set_tenant_id
    BEFORE INSERT ON document_entities
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- entity_relationships
CREATE INDEX IF NOT EXISTS idx_ai_entity_relationships_tenant_id ON entity_relationships (tenant_id);

ALTER TABLE entity_relationships FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_entity_relationships_tenant ON entity_relationships TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_entity_relationships_set_tenant_id
    BEFORE INSERT ON entity_relationships
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- providers
CREATE INDEX IF NOT EXISTS idx_ai_providers_tenant_id ON providers (tenant_id);

ALTER TABLE providers FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_providers_tenant ON providers TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_providers_set_tenant_id
    BEFORE INSERT ON providers
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- chatbots
CREATE INDEX IF NOT EXISTS idx_ai_chatbots_tenant_id ON chatbots (tenant_id);

ALTER TABLE chatbots FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_chatbots_tenant ON chatbots TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_chatbots_set_tenant_id
    BEFORE INSERT ON chatbots
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- chatbot_knowledge_bases
CREATE INDEX IF NOT EXISTS idx_ai_chatbot_knowledge_bases_tenant_id ON chatbot_knowledge_bases (tenant_id);

ALTER TABLE chatbot_knowledge_bases FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_chatbot_knowledge_bases_tenant ON chatbot_knowledge_bases TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_chatbot_knowledge_bases_set_tenant_id
    BEFORE INSERT ON chatbot_knowledge_bases
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- conversations
CREATE INDEX IF NOT EXISTS idx_ai_conversations_tenant_id ON conversations (tenant_id);

ALTER TABLE conversations FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_conversations_tenant ON conversations TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_conversations_set_tenant_id
    BEFORE INSERT ON conversations
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- messages
CREATE INDEX IF NOT EXISTS idx_ai_messages_tenant_id ON messages (tenant_id);

ALTER TABLE messages FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_messages_tenant ON messages TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_messages_set_tenant_id
    BEFORE INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- query_audit_log
CREATE INDEX IF NOT EXISTS idx_ai_query_audit_log_tenant_id ON query_audit_log (tenant_id);

ALTER TABLE query_audit_log FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_query_audit_log_tenant ON query_audit_log TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_query_audit_log_set_tenant_id
    BEFORE INSERT ON query_audit_log
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- retrieval_log
CREATE INDEX IF NOT EXISTS idx_ai_retrieval_log_tenant_id ON retrieval_log (tenant_id);

ALTER TABLE retrieval_log FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_retrieval_log_tenant ON retrieval_log TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_retrieval_log_set_tenant_id
    BEFORE INSERT ON retrieval_log
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- table_export_sync_configs
CREATE INDEX IF NOT EXISTS idx_ai_table_export_sync_configs_tenant_id ON table_export_sync_configs (tenant_id);

ALTER TABLE table_export_sync_configs FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_table_export_sync_configs_tenant ON table_export_sync_configs TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_table_export_sync_configs_set_tenant_id
    BEFORE INSERT ON table_export_sync_configs
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- user_chatbot_usage
CREATE INDEX IF NOT EXISTS idx_ai_user_chatbot_usage_tenant_id ON user_chatbot_usage (tenant_id);

ALTER TABLE user_chatbot_usage FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_user_chatbot_usage_tenant ON user_chatbot_usage TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_user_chatbot_usage_set_tenant_id
    BEFORE INSERT ON user_chatbot_usage
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- user_provider_preferences
CREATE INDEX IF NOT EXISTS idx_ai_user_provider_preferences_tenant_id ON user_provider_preferences (tenant_id);

ALTER TABLE user_provider_preferences FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_user_provider_preferences_tenant ON user_provider_preferences TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_user_provider_preferences_set_tenant_id
    BEFORE INSERT ON user_provider_preferences
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- user_quotas
CREATE INDEX IF NOT EXISTS idx_ai_user_quotas_tenant_id ON user_quotas (tenant_id);

ALTER TABLE user_quotas FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_user_quotas_tenant ON user_quotas TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_user_quotas_set_tenant_id
    BEFORE INSERT ON user_quotas
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

--
-- Cross-schema FKs moved to post-schema-fks.sql
-- knowledge_bases_created_by_fkey, knowledge_bases_owner_id_fkey, documents_created_by_fkey,
-- documents_owner_id_fkey, documents_storage_object_id_fkey, document_permissions_granted_by_fkey,
-- document_permissions_user_id_fkey, knowledge_base_permissions_granted_by_fkey,
-- knowledge_base_permissions_user_id_fkey, providers_created_by_fkey, chatbots_created_by_fkey,
-- conversations_user_id_fkey, query_audit_log_user_id_fkey, retrieval_log_user_id_fkey,
-- user_chatbot_usage_user_id_fkey, user_provider_preferences_user_id_fkey, user_quotas_user_id_fkey
--

--
-- Name: chatbot_kb_link_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER chatbot_kb_link_updated_at
    BEFORE UPDATE ON chatbot_knowledge_bases
    FOR EACH ROW
    EXECUTE FUNCTION update_chatbot_kb_link_updated_at();

--
-- Name: chunks_update_counts; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER chunks_update_counts
    AFTER INSERT OR DELETE ON chunks
    FOR EACH ROW
    EXECUTE FUNCTION update_knowledge_base_chunk_count();

--
-- Name: documents_set_owner; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER documents_set_owner
    BEFORE INSERT ON documents
    FOR EACH ROW
    EXECUTE FUNCTION set_document_owner();

--
-- Name: documents_update_kb_counts; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER documents_update_kb_counts
    AFTER INSERT OR DELETE ON documents
    FOR EACH ROW
    EXECUTE FUNCTION update_knowledge_base_counts();

--
-- Name: documents_update_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER documents_update_updated_at
    BEFORE UPDATE ON documents
    FOR EACH ROW
    EXECUTE FUNCTION update_knowledge_base_updated_at();

--
-- Name: entities_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER entities_updated_at
    BEFORE UPDATE ON entities
    FOR EACH ROW
    EXECUTE FUNCTION update_entities_updated_at();

--
-- Name: knowledge_bases_update_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER knowledge_bases_update_updated_at
    BEFORE UPDATE ON knowledge_bases
    FOR EACH ROW
    EXECUTE FUNCTION update_knowledge_base_updated_at();

--
-- Name: set_chunk_user_id_trigger; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER set_chunk_user_id_trigger
    BEFORE INSERT ON chunks
    FOR EACH ROW
    EXECUTE FUNCTION set_chunk_user_id();

--
-- Name: set_document_user_id_trigger; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER set_document_user_id_trigger
    BEFORE INSERT ON documents
    FOR EACH ROW
    EXECUTE FUNCTION set_document_user_id();

--
-- Name: trigger_update_table_export_sync_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER trigger_update_table_export_sync_updated_at
    BEFORE UPDATE ON table_export_sync_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_table_export_sync_updated_at();

--
-- Name: chatbot_knowledge_bases; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE chatbot_knowledge_bases TO authenticated;

--
-- Name: chatbot_knowledge_bases; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE chatbot_knowledge_bases TO {{APP_USER}};

--
-- Name: chatbot_knowledge_bases; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: chatbot_knowledge_bases; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE chatbot_knowledge_bases TO service_role;

--
-- Name: chatbots; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE chatbots TO authenticated;

--
-- Name: chatbots; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE chatbots TO {{APP_USER}};

--
-- Name: chatbots; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: chatbots; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE chatbots TO service_role;

--
-- Name: chunks; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE chunks TO authenticated;

--
-- Name: chunks; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE chunks TO {{APP_USER}};

--
-- Name: chunks; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: chunks; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE chunks TO service_role;

--
-- Name: conversations; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE conversations TO authenticated;

--
-- Name: conversations; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE conversations TO {{APP_USER}};

--
-- Name: conversations; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: conversations; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE conversations TO service_role;

--
-- Name: document_entities; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE document_entities TO authenticated;

--
-- Name: document_entities; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE document_entities TO {{APP_USER}};

--
-- Name: document_entities; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: document_entities; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE document_entities TO service_role;

--
-- Name: document_permissions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE document_permissions TO authenticated;

--
-- Name: document_permissions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE document_permissions TO {{APP_USER}};

--
-- Name: document_permissions; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: document_permissions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE document_permissions TO service_role;

--
-- Name: documents; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE documents TO authenticated;

--
-- Name: documents; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE documents TO {{APP_USER}};

--
-- Name: documents; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: documents; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE documents TO service_role;

--
-- Name: entities; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE entities TO authenticated;

--
-- Name: entities; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE entities TO {{APP_USER}};

--
-- Name: entities; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: entities; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE entities TO service_role;

--
-- Name: entity_relationships; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE entity_relationships TO authenticated;

--
-- Name: entity_relationships; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE entity_relationships TO {{APP_USER}};

--
-- Name: entity_relationships; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: entity_relationships; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE entity_relationships TO service_role;

--
-- Name: knowledge_base_permissions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE knowledge_base_permissions TO authenticated;

--
-- Name: knowledge_base_permissions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE knowledge_base_permissions TO {{APP_USER}};

--
-- Name: knowledge_base_permissions; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: knowledge_base_permissions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE knowledge_base_permissions TO service_role;

--
-- Name: knowledge_bases; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE knowledge_bases TO authenticated;

--
-- Name: knowledge_bases; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE knowledge_bases TO {{APP_USER}};

--
-- Name: knowledge_bases; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: knowledge_bases; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE knowledge_bases TO service_role;

--
-- Name: messages; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE messages TO authenticated;

--
-- Name: messages; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE messages TO {{APP_USER}};

--
-- Name: messages; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: messages; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE messages TO service_role;

--
-- Name: providers; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE providers TO authenticated;

--
-- Name: providers; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE providers TO {{APP_USER}};

--
-- Name: providers; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: providers; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE providers TO service_role;

--
-- Name: query_audit_log; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE query_audit_log TO authenticated;

--
-- Name: query_audit_log; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE query_audit_log TO {{APP_USER}};

--
-- Name: query_audit_log; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: query_audit_log; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE query_audit_log TO service_role;

--
-- Name: retrieval_log; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE retrieval_log TO authenticated;

--
-- Name: retrieval_log; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE retrieval_log TO {{APP_USER}};

--
-- Name: retrieval_log; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: retrieval_log; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE retrieval_log TO service_role;

--
-- Name: table_export_sync_configs; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE table_export_sync_configs TO authenticated;

--
-- Name: table_export_sync_configs; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE table_export_sync_configs TO {{APP_USER}};

--
-- Name: table_export_sync_configs; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: table_export_sync_configs; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE table_export_sync_configs TO service_role;

--
-- Name: user_chatbot_usage; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE user_chatbot_usage TO authenticated;

--
-- Name: user_chatbot_usage; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE user_chatbot_usage TO {{APP_USER}};

--
-- Name: user_chatbot_usage; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: user_chatbot_usage; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE user_chatbot_usage TO service_role;

--
-- Name: user_provider_preferences; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE user_provider_preferences TO authenticated;

--
-- Name: user_provider_preferences; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE user_provider_preferences TO {{APP_USER}};

--
-- Name: user_provider_preferences; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: user_provider_preferences; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE user_provider_preferences TO service_role;

--
-- Name: user_quotas; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE user_quotas TO authenticated;

--
-- Name: user_quotas; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE user_quotas TO {{APP_USER}};

--
-- Name: user_quotas; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: user_quotas; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE user_quotas TO service_role;

--
-- Name: tenant_service grants; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT USAGE ON SCHEMA ai TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE knowledge_bases TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE knowledge_base_permissions TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE documents TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE document_permissions TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE chunks TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE entities TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE document_entities TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE entity_relationships TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE providers TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE chatbots TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE chatbot_knowledge_bases TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE conversations TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE messages TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE query_audit_log TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE retrieval_log TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE table_export_sync_configs TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE user_chatbot_usage TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE user_provider_preferences TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE user_quotas TO tenant_service;

-- ============================================================================
-- Tool Integrations
-- ============================================================================
-- Non-LLM external services that chatbot specialists can call. Currently
-- supports web_search (Tavily, Brave, etc.) and fetch_url (single-page
-- extract). Schema is provider-agnostic so new integrations (code runner,
-- GitHub, etc.) can be added without restructuring.
--
-- Secrets in `config` (e.g., api_key) MUST be encrypted at the application
-- layer via internal/ai/integrations/crypto.go using the master
-- EncryptionKey. Read paths auto-detect legacy plaintext values and treat
-- them as plaintext for forward compatibility with the lazy encryption
-- migration on ai.providers.

CREATE TABLE IF NOT EXISTS tool_integrations (
    id uuid DEFAULT gen_random_uuid(),
    name text NOT NULL,
    integration_type text NOT NULL,
    provider text NOT NULL,
    config jsonb DEFAULT '{}' NOT NULL,
    enabled boolean DEFAULT true,
    is_default boolean DEFAULT false,
    last_tested_at timestamptz,
    last_test_status text,
    last_test_error text,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    created_by uuid,
    tenant_id uuid,
    CONSTRAINT tool_integrations_pkey PRIMARY KEY (id),
    CONSTRAINT tool_integrations_name_key UNIQUE (name),
    CONSTRAINT tool_integrations_type_check
        CHECK (integration_type IN ('web_search','fetch_url')),
    CONSTRAINT tool_integrations_provider_check
        CHECK (provider IN ('tavily','brave','jina','singlefetch'))
);

CREATE INDEX IF NOT EXISTS idx_ai_tool_integrations_tenant_id ON tool_integrations (tenant_id);
CREATE INDEX IF NOT EXISTS idx_ai_tool_integrations_enabled ON tool_integrations (enabled);
CREATE INDEX IF NOT EXISTS idx_ai_tool_integrations_type ON tool_integrations (integration_type);

-- One default per (integration_type, tenant_id). Postgres partial unique index
-- ensures exactly one row per type per tenant can have is_default=true.
-- Tenant_id NULL collides with other NULLs by default which is what we want
-- (only one instance-wide default when no tenant context).
CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_tool_integrations_single_default
    ON tool_integrations (integration_type, tenant_id)
    WHERE (is_default = true);

ALTER TABLE tool_integrations ENABLE ROW LEVEL SECURITY;
ALTER TABLE tool_integrations FORCE ROW LEVEL SECURITY;

CREATE POLICY tool_integrations_tenant ON tool_integrations TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER ai_tool_integrations_set_tenant_id
    BEFORE INSERT ON tool_integrations
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

COMMENT ON TABLE tool_integrations IS
    'External tool integrations (web search, URL fetch) consumed by chatbot specialist agents';
COMMENT ON COLUMN tool_integrations.config IS
    'Provider-specific config JSON. Secret fields (api_key) MUST be encrypted at application layer.';
COMMENT ON COLUMN tool_integrations.last_test_status IS
    'Result of the most recent test-connection call: ok, failed, or NULL if never tested';

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE tool_integrations TO authenticated;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE tool_integrations TO tenant_service;

