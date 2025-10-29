-- Paper Types Table
CREATE TABLE IF NOT EXISTS paper_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    display_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Paper Images Table
CREATE TABLE IF NOT EXISTS paper_images (
    id SERIAL PRIMARY KEY,
    type_id INTEGER NOT NULL REFERENCES paper_types(id) ON DELETE CASCADE,
    image_url TEXT NOT NULL,
    display_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_paper_images_type_id ON paper_images(type_id);
CREATE INDEX IF NOT EXISTS idx_paper_images_display_order ON paper_images(display_order);
CREATE INDEX IF NOT EXISTS idx_paper_types_display_order ON paper_types(display_order);

-- Sample data
INSERT INTO paper_types (name, display_order, is_active) VALUES
('Myanmar News', 1, true),
('Thailand News', 2, true),
('International', 3, true)
ON CONFLICT (name) DO NOTHING;
