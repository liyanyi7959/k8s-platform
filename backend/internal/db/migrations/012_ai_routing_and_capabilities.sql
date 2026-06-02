ALTER TABLE ai_providers
  ADD COLUMN vendor_code VARCHAR(32) NOT NULL DEFAULT '' AFTER provider_type,
  ADD COLUMN auth_scheme VARCHAR(32) NOT NULL DEFAULT 'bearer' AFTER base_url;

ALTER TABLE ai_models
  ADD COLUMN supports_streaming TINYINT(1) NOT NULL DEFAULT 0 AFTER supports_vision,
  ADD COLUMN supports_reasoning TINYINT(1) NOT NULL DEFAULT 0 AFTER supports_streaming,
  ADD COLUMN supports_structured_output TINYINT(1) NOT NULL DEFAULT 0 AFTER supports_reasoning,
  ADD COLUMN supports_image_generation TINYINT(1) NOT NULL DEFAULT 0 AFTER supports_structured_output,
  ADD COLUMN context_window INT NOT NULL DEFAULT 0 AFTER max_output_tokens;

CREATE TABLE IF NOT EXISTS ai_route_settings (
  id                                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  default_chat_model_id             BIGINT UNSIGNED NULL,
  default_diagnose_model_id         BIGINT UNSIGNED NULL,
  default_vision_model_id           BIGINT UNSIGNED NULL,
  default_image_generation_model_id BIGINT UNSIGNED NULL,
  default_fallback_provider_id      BIGINT UNSIGNED NULL,
  routing_strategy                  VARCHAR(32) NOT NULL DEFAULT 'priority_first',
  allow_fallback                    TINYINT(1) NOT NULL DEFAULT 1,
  meta_json                         JSON NULL,
  created_at                        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at                        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_ai_route_settings_chat_model FOREIGN KEY (default_chat_model_id) REFERENCES ai_models(id) ON DELETE SET NULL,
  CONSTRAINT fk_ai_route_settings_diagnose_model FOREIGN KEY (default_diagnose_model_id) REFERENCES ai_models(id) ON DELETE SET NULL,
  CONSTRAINT fk_ai_route_settings_vision_model FOREIGN KEY (default_vision_model_id) REFERENCES ai_models(id) ON DELETE SET NULL,
  CONSTRAINT fk_ai_route_settings_image_model FOREIGN KEY (default_image_generation_model_id) REFERENCES ai_models(id) ON DELETE SET NULL,
  CONSTRAINT fk_ai_route_settings_fallback_provider FOREIGN KEY (default_fallback_provider_id) REFERENCES ai_providers(id) ON DELETE SET NULL
);

INSERT INTO ai_route_settings (
  id,
  default_chat_model_id,
  default_diagnose_model_id,
  default_vision_model_id,
  default_image_generation_model_id,
  default_fallback_provider_id,
  routing_strategy,
  allow_fallback,
  meta_json
)
SELECT
  1,
  NULL,
  NULL,
  NULL,
  NULL,
  NULL,
  'priority_first',
  1,
  NULL
WHERE NOT EXISTS (
  SELECT 1 FROM ai_route_settings WHERE id = 1
);
