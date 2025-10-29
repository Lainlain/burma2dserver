-- Create threed table for 3D lottery results
CREATE TABLE IF NOT EXISTS threed (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    result VARCHAR(3) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index on date for faster queries
CREATE INDEX IF NOT EXISTS idx_threed_date ON threed(date DESC);

-- Insert sample data
INSERT INTO threed (date, result) VALUES 
('2025-10-16', '696'),
('2025-10-01', '978'),
('2025-09-16', '646'),
('2025-09-01', '356'),
('2025-08-16', '865'),
('2025-08-01', '852'),
('2025-07-16', '324'),
('2025-07-01', '246'),
('2025-06-16', '392'),
('2025-06-01', '352'),
('2025-05-31', '309'),
('2025-05-16', '402')
ON CONFLICT (date) DO NOTHING;
