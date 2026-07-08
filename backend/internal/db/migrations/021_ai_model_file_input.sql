ALTER TABLE ai_models
  ADD COLUMN supports_file_input TINYINT(1) NOT NULL DEFAULT 0 AFTER supports_image_generation;
