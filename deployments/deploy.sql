-- ============================================
-- MANGA DISCORD-LIKE FORUM - PostgreSQL SCHEMA
-- Optimized, Minimal, Production-Ready
-- ============================================

-- Note (Neon-friendly): no CREATE EXTENSION statements here.
-- This schema uses TEXT IDs (app-generated), so extensions are not required.

-- Drop tables if exist (for clean migrations)
DROP TABLE IF EXISTS manga_stats CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS activity_feed CASCADE;
DROP TABLE IF EXISTS chat_messages CASCADE;
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS manga_genres CASCADE;
DROP TABLE IF EXISTS genres CASCADE;
DROP TABLE IF EXISTS manga CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- ============================================
-- 1. USERS (AUTH + ROLE)
-- ============================================

CREATE TABLE users (
  id TEXT PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'user'
    CHECK (role IN ('user', 'moderator', 'admin')),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_role ON users(role);

-- ============================================
-- 2. MANGA (CORE CONTENT)
-- ============================================

CREATE TABLE manga (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT,
  cover_url TEXT,
  status TEXT DEFAULT 'ongoing',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_manga_title ON manga(title);
CREATE INDEX idx_manga_status ON manga(status);
CREATE INDEX idx_manga_created_at ON manga(created_at DESC);
CREATE INDEX idx_manga_updated_at ON manga(updated_at DESC);

-- ============================================
-- 3. GENRES + MANGA_GENRES (BROWSE)
-- ============================================

CREATE TABLE genres (
  id TEXT PRIMARY KEY,
  name TEXT UNIQUE NOT NULL
);

CREATE INDEX idx_genres_name ON genres(name);

CREATE TABLE manga_genres (
  manga_id TEXT,
  genre_id TEXT,
  PRIMARY KEY (manga_id, genre_id),
  FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE,
  FOREIGN KEY (genre_id) REFERENCES genres(id) ON DELETE CASCADE
);

CREATE INDEX idx_manga_genres_manga_id ON manga_genres(manga_id);
CREATE INDEX idx_manga_genres_genre_id ON manga_genres(genre_id);

-- ============================================
-- 4. FULL-TEXT SEARCH (gRPC)
-- ============================================

-- Add tsvector column for full-text search
ALTER TABLE manga ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Create index for full-text search
CREATE INDEX idx_manga_search_vector ON manga USING GIN(search_vector);

-- Function to update search vector
CREATE OR REPLACE FUNCTION manga_search_vector_update() RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update search vector
DROP TRIGGER IF EXISTS manga_search_vector_trigger ON manga;
CREATE TRIGGER manga_search_vector_trigger
  BEFORE INSERT OR UPDATE ON manga
  FOR EACH ROW
  EXECUTE FUNCTION manga_search_vector_update();

-- ============================================
-- 5. COMMENTS (COMMENT + LIKE)
-- ============================================

CREATE TABLE comments (
  id TEXT PRIMARY KEY,
  manga_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  content TEXT NOT NULL,
  like_count INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_comments_manga_id ON comments(manga_id);
CREATE INDEX idx_comments_user_id ON comments(user_id);
CREATE INDEX idx_comments_created_at ON comments(created_at DESC);
CREATE INDEX idx_comments_like_count ON comments(like_count DESC);

-- ============================================
-- 6. CHAT MESSAGES (WEBSOCKET)
-- ============================================

CREATE TABLE chat_messages (
  id TEXT PRIMARY KEY,
  manga_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_chat_messages_manga_id ON chat_messages(manga_id);
CREATE INDEX idx_chat_messages_user_id ON chat_messages(user_id);
CREATE INDEX idx_chat_messages_created_at ON chat_messages(created_at DESC);

-- ============================================
-- 7. ACTIVITY FEED (HOME)
-- ============================================

CREATE TABLE activity_feed (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL
    CHECK (type IN ('comment', 'chat', 'manga_update')),
  user_id TEXT,
  manga_id TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_activity_feed_type ON activity_feed(type);
CREATE INDEX idx_activity_feed_user_id ON activity_feed(user_id);
CREATE INDEX idx_activity_feed_manga_id ON activity_feed(manga_id);
CREATE INDEX idx_activity_feed_created_at ON activity_feed(created_at DESC);

-- ============================================
-- 8. NOTIFICATIONS (UDP BROADCAST LOG)
-- ============================================

CREATE TABLE notifications (
  id TEXT PRIMARY KEY,
  message TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);

-- ============================================
-- 9. MANGA STATS (TCP AGGREGATION)
-- ============================================

CREATE TABLE manga_stats (
  manga_id TEXT PRIMARY KEY,
  comment_count INTEGER DEFAULT 0,
  like_count INTEGER DEFAULT 0,
  chat_count INTEGER DEFAULT 0,
  weekly_score INTEGER DEFAULT 0,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE
);

CREATE INDEX idx_manga_stats_weekly_score ON manga_stats(weekly_score DESC);
CREATE INDEX idx_manga_stats_comment_count ON manga_stats(comment_count DESC);
CREATE INDEX idx_manga_stats_updated_at ON manga_stats(updated_at DESC);

-- ============================================
-- 10. SEED INITIAL DATA
-- ============================================

-- Seed genres
INSERT INTO genres (id, name) VALUES
  ('action', 'Action'),
  ('adventure', 'Adventure'),
  ('comedy', 'Comedy'),
  ('drama', 'Drama'),
  ('fantasy', 'Fantasy'),
  ('horror', 'Horror'),
  ('mystery', 'Mystery'),
  ('romance', 'Romance'),
  ('sci-fi', 'Sci-Fi'),
  ('slice-of-life', 'Slice of Life'),
  ('sports', 'Sports'),
  ('supernatural', 'Supernatural')
ON CONFLICT (id) DO NOTHING;

-- Create default admin user (password: admin123)
-- Note: This should be changed in production
INSERT INTO users (id, username, password_hash, role, created_at) VALUES
  ('admin-001', 'we!xu', '$2a$10$ffzzu4GxSD9z0eEa/wNmK.8JfFNECDyzFREQH1qV6RgQ8lxtqT3MW', 'admin', CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

-- ============================================
-- END OF SCHEMA
-- ============================================
