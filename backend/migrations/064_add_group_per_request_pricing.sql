-- 按次计费配置：模型模式 -> 固定单价（USD）
ALTER TABLE groups ADD COLUMN IF NOT EXISTS per_request_pricing JSONB;
COMMENT ON COLUMN groups.per_request_pricing IS '按次计费配置：模型模式(支持通配符) -> 每次请求固定价格(USD)';