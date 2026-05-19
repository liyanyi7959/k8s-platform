CREATE TABLE IF NOT EXISTS kb_doc_tags (
  doc_node_id BIGINT UNSIGNED NOT NULL,
  tag VARCHAR(64) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (doc_node_id, tag),
  CONSTRAINT fk_kb_doc_tags_doc FOREIGN KEY (doc_node_id) REFERENCES kb_docs(node_id)
);
