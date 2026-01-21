-- ============================================
-- MANGA DISCORD-LIKE FORUM - SAMPLE DATA SEED
-- Organized for protocol testing scenarios
-- ============================================

-- ============================================
-- 1. USERS (Admin + Test Users)
-- ============================================

-- Moderator users (password: moderator123)
INSERT INTO users (id, username, password_hash, role) VALUES
  ('mod-001', 'moderator_jin', '$2a$10$okkzW2mAPY.8MjxsU48ZBe5HNvdgGUyqis/4OVQcQ4FKm2zKUXmom', 'moderator'),
  ('mod-002', 'moderator_leo', '$2a$10$okkzW2mAPY.8MjxsU48ZBe5HNvdgGUyqis/4OVQcQ4FKm2zKUXmom', 'moderator');

INSERT INTO users (id, username, password_hash, role) VALUES
  ('user-001', 'alice', '$2a$10$HBz.KzHkEPdh0wbdtJr0KegH2DHz8PAtMDBOtBGb1TkYRFihSYV0e', 'user'),
  ('user-002', 'bob', '$2a$10$HBz.KzHkEPdh0wbdtJr0KegH2DHz8PAtMDBOtBGb1TkYRFihSYV0e', 'user'),
  ('user-003', 'charlie', '$2a$10$HBz.KzHkEPdh0wbdtJr0KegH2DHz8PAtMDBOtBGb1TkYRFihSYV0e', 'user'),
  ('user-004', 'diana', '$2a$10$HBz.KzHkEPdh0wbdtJr0KegH2DHz8PAtMDBOtBGb1TkYRFihSYV0e', 'user'),
  ('user-005', 'edward', '$2a$10$HBz.KzHkEPdh0wbdtJr0KegH2DHz8PAtMDBOtBGb1TkYRFihSYV0e', 'user'),
  ('user-006', 'fiona', '$2a$10$HBz.KzHkEPdh0wbdtJr0KegH2DHz8PAtMDBOtBGb1TkYRFihSYV0e', 'user'),
  ('user-007', 'george', '$2a$10$HBz.KzHkEPdh0wbdtJr0KegH2DHz8PAtMDBOtBGb1TkYRFihSYV0e', 'user'),
  ('user-008', 'hannah', '$2a$10$HBz.KzHkEPdh0wbdtJr0KegH2DHz8PAtMDBOtBGb1TkYRFihSYV0e', 'user');

-- ============================================
-- 2. MANGA TITLES (Curated for testing scenarios)
-- ============================================

-- Hot/Popular manga (high activity)
INSERT INTO manga (id, title, description, cover_url, status) VALUES
  ('manga-001', 'One Piece', 'The story of Monkey D. Luffy and his quest to become the Pirate King by finding the legendary treasure One Piece.', 'https://example.com/covers/onepiece.jpg', 'ongoing'),
  ('manga-002', 'Jujutsu Kaisen', 'Yuji Itadori is an unnaturally fit high school student who joins the Occult Research Club. When they accidentally unseal a cursed object, Yuji swallows it to protect his friends, becoming host to the powerful Curse, Sukuna.', 'https://example.com/covers/jjk.jpg', 'ongoing'),
  ('manga-003', 'Chainsaw Man', 'Denji has a simple dream—to live a happy and peaceful life, spending time with a girl he likes. However, this is a far cry from reality, as Denji is forced by the yakuza into killing devils in order to pay off his crushing debts.', 'https://example.com/covers/chainsawman.jpg', 'ongoing');

-- Mid-tier manga (moderate activity)
INSERT INTO manga (id, title, description, cover_url, status) VALUES
  ('manga-004', 'Kaguya-sama: Love is War', 'Two geniuses, Miyuki Shirogane and Kaguya Shinomiya, are the top students at Shuchiin Academy. Both are in love with each other, but neither is willing to confess since doing so would be a sign of weakness.', 'https://example.com/covers/kaguya.jpg', 'completed'),
  ('manga-005', 'Spy x Family', 'A spy on an undercover mission gets married and adopts a child as part of his cover. His wife and daughter have secrets of their own, and all three must strive to keep their true identities hidden from each other.', 'https://example.com/covers/spyfamily.jpg', 'ongoing'),
  ('manga-006', 'Blue Lock', 'After a disastrous defeat at the 2018 World Cup, Japan''s team struggles to regroup. The Japanese Football Association hires enigmatic coach Jinpachi Ego to achieve their dream—winning the World Cup.', 'https://example.com/covers/bluelock.jpg', 'ongoing');

-- Niche manga (low activity)
INSERT INTO manga (id, title, description, cover_url, status) VALUES
  ('manga-007', 'Frieren: Beyond Journey''s End', 'After the party of heroes defeated the Demon King, they restored peace to the land and returned to lives of solitude. The elven mage Frieren, nearly immortal, did not realize how much her brief time with them came to mean to her.', 'https://example.com/covers/frieren.jpg', 'ongoing'),
  ('manga-008', 'Delicious in Dungeon', 'After his sister is devoured by a dragon, Laios and his party must venture into the very same dungeon to rescue her before she is digested. The problem? They''re broke and starving.', 'https://example.com/covers/dungeon.jpg', 'ongoing'),
  ('manga-009', 'Oshi no Ko', 'Gorou is a gynecologist who is a fan of the idol Ai Hoshino. When Ai comes to his office pregnant and afraid, he takes her under his wing. But one day, Ai—and Gorou—are murdered, and Gorou is reborn as Ai''s son, Aquamarine.', 'https://example.com/covers/oshi.jpg', 'ongoing'),
  ('manga-010', 'Hell''s Paradise', 'Gabimaru the Hollow, a ninja of Iwagakure Village known for being cold-hearted and killing without remorse, is on death row. There he is approached by an executioner, Yamada Asaemon Sagiri, about a pardon from the Shogunate for a dangerous mission to an island south of Japan.', 'https://example.com/covers/hellparadise.jpg', 'ongoing');

-- Update search vectors for all manga
UPDATE manga SET search_vector = 
  setweight(to_tsvector('english', COALESCE(title, '')), 'A') ||
  setweight(to_tsvector('english', COALESCE(description, '')), 'B');

-- ============================================
-- 3. GENRES ASSIGNMENT (Many-to-Many)
-- ============================================

-- One Piece genres
INSERT INTO manga_genres (manga_id, genre_id) VALUES
  ('manga-001', 'action'),
  ('manga-001', 'adventure'),
  ('manga-001', 'comedy'),
  ('manga-001', 'fantasy');

-- Jujutsu Kaisen genres
INSERT INTO manga_genres (manga_id, genre_id) VALUES
  ('manga-002', 'action'),
  ('manga-002', 'supernatural'),
  ('manga-002', 'horror');

-- Chainsaw Man genres
INSERT INTO manga_genres (manga_id, genre_id) VALUES
  ('manga-003', 'action'),
  ('manga-003', 'supernatural'),
  ('manga-003', 'horror'),
  ('manga-003', 'comedy');

-- Kaguya-sama genres
INSERT INTO manga_genres (manga_id, genre_id) VALUES
  ('manga-004', 'comedy'),
  ('manga-004', 'romance'),
  ('manga-004', 'psychological');

-- Spy x Family genres
INSERT INTO manga_genres (manga_id, genre_id) VALUES
  ('manga-005', 'action'),
  ('manga-005', 'comedy'),
  ('manga-005', 'slice-of-life');

-- Blue Lock genres
INSERT INTO manga_genres (manga_id, genre_id) VALUES
  ('manga-006', 'sports'),
  ('manga-006', 'drama'),
  ('manga-006', 'psychological');

-- Frieren genres
INSERT INTO manga_genres (manga_id, genre_id) VALUES
  ('manga-007', 'adventure'),
  ('manga-007', 'fantasy'),
  ('manga-007', 'drama'),
  ('manga-007', 'slice-of-life');

-- Delicious in Dungeon genres
INSERT INTO manga_genres (manga_id, genre_id) VALUES
  ('manga-008', 'adventure'),
  ('manga-008', 'fantasy'),
  ('manga-008', 'comedy');

-- Oshi no Ko genres
INSERT INTO manga_genres (manga_id, genre_id) VALUES
  ('manga-009', 'drama'),
  ('manga-009', 'psychological'),
  ('manga-009', 'supernatural');

-- Hell's Paradise genres
INSERT INTO manga_genres (manga_id, genre_id) VALUES
  ('manga-010', 'action'),
  ('manga-010', 'adventure'),
  ('manga-010', 'supernatural'),
  ('manga-010', 'horror');

-- ============================================
-- 4. COMMENTS (For HTTP API testing)
-- ============================================

-- One Piece comments (high engagement)
INSERT INTO comments (id, manga_id, user_id, content, like_count) VALUES
  ('comment-001', 'manga-001', 'user-003', 'Luffy finally got his gear 5 transformation! This is insane!', 45),
  ('comment-002', 'manga-001', 'user-001', 'The animation in the anime adaptation is absolutely breathtaking. Can''t wait for more chapters!', 32),
  ('comment-003', 'manga-001', 'user-006', 'I think the mystery behind Joy Boy is going to be the key to everything. Eiichiro Oda has been planning this for decades.', 28),
  ('comment-004', 'manga-001', 'user-002', 'Zoro finally got his 3-sword style perfected. About time!', 19),
  ('comment-005', 'manga-001', 'user-007', 'The Straw Hat Pirates crew dynamics are what make this series so special. Each character has their own dreams and motivations.', 15);

-- Jujutsu Kaisen comments
INSERT INTO comments (id, manga_id, user_id, content, like_count) VALUES
  ('comment-006', 'manga-002', 'user-004', 'Gojo Satoru is the strongest jujutsu sorcerer for a reason. His domain expansion is terrifying!', 38),
  ('comment-007', 'manga-002', 'user-001', 'The manga is so much better than the anime. The manga artist''s art style is incredible.', 24),
  ('comment-008', 'manga-002', 'user-003', 'Sukuna vs Mahito fight was epic! I can''t believe how powerful Sukuna has become.', 21),
  ('comment-009', 'manga-002', 'user-005', 'I''m worried about Yuji''s future. The way he''s being manipulated by both sides is heartbreaking.', 17),
  ('comment-010', 'manga-002', 'user-002', 'Megumi''s growth as a sorcerer has been amazing to watch. His domain expansion potential is huge!', 14);

-- Chainsaw Man comments
INSERT INTO comments (id, manga_id, user_id, content, like_count) VALUES
  ('comment-011', 'manga-003', 'user-004', 'Denji''s character development is so raw and real. He''s not your typical hero.', 35),
  ('comment-012', 'manga-003', 'user-008', 'The horror elements in this series are genuinely disturbing. Fujimoto doesn''t hold back.', 29),
  ('comment-013', 'manga-003', 'user-001', 'Power''s death hit me harder than any other character death in recent manga history.', 26),
  ('comment-014', 'manga-003', 'user-003', 'The way Fujimoto blends comedy with extreme violence is masterful. No other manga does this as well.', 22),
  ('comment-015', 'manga-003', 'user-006', 'I''m excited for the second part! The setup for the new characters looks promising.', 18);

-- ============================================
-- 5. CHAT MESSAGES (For WebSocket testing)
-- ============================================

-- One Piece chat room (active discussion)
INSERT INTO chat_messages (id, manga_id, user_id, content) VALUES
  ('chat-001', 'manga-001', 'user-003', 'Chapter 1095 just dropped! Luffy vs Kaido part 2 incoming!'),
  ('chat-002', 'manga-001', 'user-001', 'Did you see that new technique Luffy used? It was called "Gear 5th: Nika Form"'),
  ('chat-003', 'manga-001', 'user-007', 'I''m rewatching the Marineford arc while waiting for new chapters. Still emotional.'),
  ('chat-004', 'manga-001', 'user-002', 'Zoro needs more screen time. He''s been training but we haven''t seen much of his growth lately.'),
  ('chat-005', 'manga-001', 'user-003', 'Theory: Shanks is going to be the final villain. His connection to the World Government is too suspicious.'),
  ('chat-006', 'manga-001', 'user-006', 'The Void Century revelations are going to shake the entire One Piece world.'),
  ('chat-007', 'manga-001', 'user-004', 'Who else is excited for the live-action season 2? They did such a great job with season 1.'),
  ('chat-008', 'manga-001', 'user-001', 'I think Sabo is going to have a major role in the final saga. His burning of the World Noble flag was just the beginning.'),
  ('chat-009', 'manga-001', 'user-005', 'The art style in the Wano arc is incredible. Oda really stepped up his game.'),
  ('chat-010', 'manga-001', 'user-003', 'Anyone else think Robin''s past is going to be crucial to understanding the Ancient Weapons?');

-- Jujutsu Kaisen chat room (moderately active)
INSERT INTO chat_messages (id, manga_id, user_id, content) VALUES
  ('chat-011', 'manga-002', 'user-004', 'The Shibuya Incident arc is the best arc in any manga I''ve ever read.'),
  ('chat-012', 'manga-002', 'user-001', 'Gojo''s seal breaking was the most hype moment of my life.'),
  ('chat-013', 'manga-002', 'user-002', 'Megumi is underrated. His shadow technique is so versatile and powerful.'),
  ('chat-014', 'manga-002', 'user-004', 'I''m worried about Yuji. His mental state after all the losses is concerning.'),
  ('chat-015', 'manga-002', 'user-003', 'The anime adaptation of the Shibuya arc was perfect. MAPPA did an amazing job.'),
  ('chat-016', 'manga-002', 'user-005', 'Sukuna''s origin story is fascinating. The connection to the Heian era is brilliant storytelling.'),
  ('chat-017', 'manga-002', 'user-004', 'Theory: Yuji and Megumi are brothers. The foreshadowing is there if you pay attention.'),
  ('chat-018', 'manga-002', 'user-001', 'Geto''s character arc is one of the most tragic in manga history. Such wasted potential.'),
  ('chat-019', 'manga-002', 'user-002', 'The Culling Game arc is confusing but I trust Gege to tie everything together.'),
  ('chat-020', 'manga-002', 'user-004', 'Choso becoming Yuji''s brother is one of the best character developments ever.');

-- Chainsaw Man chat room (active horror discussion)
INSERT INTO chat_messages (id, manga_id, user_id, content) VALUES
  ('chat-021', 'manga-003', 'user-008', 'The violence in chapter 97 was insane. Fujimoto really doesn''t care about censorship.'),
  ('chat-022', 'manga-003', 'user-004', 'Denji''s relationship with Power was the heart of part 1. Her death destroyed me.'),
  ('chat-023', 'manga-003', 'user-001', 'Makima is the most terrifying villain I''ve ever encountered in manga. Her manipulation tactics are so realistic.'),
  ('chat-024', 'manga-003', 'user-008', 'The way Fujimoto portrays mental health issues through Denji is really thoughtful despite the gore.'),
  ('chat-025', 'manga-003', 'user-003', 'I''m excited for the new characters in part 2. Yoru looks interesting.'),
  ('chat-026', 'manga-003', 'user-004', 'The religious symbolism in Chainsaw Man is deep. Denji as a Christ figure, Makima as a false god, etc.'),
  ('chat-027', 'manga-003', 'user-008', 'The anime adaptation was perfect. The MAPPA team captured Fujimoto''s raw art style beautifully.'),
  ('chat-028', 'manga-003', 'user-001', 'Aki''s death was so tragic. He had so much to live for after finally finding purpose.'),
  ('chat-029', 'manga-003', 'user-004', 'The way Denji eats tomatoes to cope with trauma is such a human detail in such an inhuman world.'),
  ('chat-030', 'manga-003', 'user-008', 'I think the second part will explore Denji''s humanity more. He''s been through so much.');

-- ============================================
-- 6. ACTIVITY FEED (For homepage feed testing)
-- ============================================

-- Recent comment activities
INSERT INTO activity_feed (id, type, user_id, manga_id) VALUES
  ('activity-001', 'comment', 'user-003', 'manga-001'),
  ('activity-002', 'comment', 'user-001', 'manga-001'),
  ('activity-003', 'comment', 'user-004', 'manga-002'),
  ('activity-004', 'comment', 'user-004', 'manga-003'),
  ('activity-005', 'comment', 'user-008', 'manga-003');

-- Recent chat activities
INSERT INTO activity_feed (id, type, user_id, manga_id) VALUES
  ('activity-006', 'chat', 'user-003', 'manga-001'),
  ('activity-007', 'chat', 'user-004', 'manga-002'),
  ('activity-008', 'chat', 'user-008', 'manga-003');

-- Manga update activities (by admin)
INSERT INTO activity_feed (id, type, user_id, manga_id) VALUES
  ('activity-009', 'manga_update', 'admin-001', 'manga-001'),
  ('activity-010', 'manga_update', 'admin-001', 'manga-002'),
  ('activity-011', 'manga_update', 'admin-001', 'manga-003'),
  ('activity-012', 'manga_update', 'admin-001', 'manga-004'),
  ('activity-013', 'manga_update', 'admin-001', 'manga-005');

-- ============================================
-- 7. MANGA STATS (For TCP stats service testing)
-- ============================================

-- High-activity manga stats
INSERT INTO manga_stats (manga_id, comment_count, like_count, chat_count, weekly_score) VALUES
  ('manga-001', 150, 320, 85, 950),  -- One Piece (very hot)
  ('manga-002', 120, 280, 75, 880),   -- Jujutsu Kaisen (hot)
  ('manga-003', 110, 260, 70, 850);   -- Chainsaw Man (hot)

-- Medium-activity manga stats
INSERT INTO manga_stats (manga_id, comment_count, like_count, chat_count, weekly_score) VALUES
  ('manga-004', 85, 190, 50, 720),    -- Kaguya-sama
  ('manga-005', 90, 210, 55, 750),    -- Spy x Family
  ('manga-006', 75, 180, 45, 680);    -- Blue Lock

-- Low-activity manga stats
INSERT INTO manga_stats (manga_id, comment_count, like_count, chat_count, weekly_score) VALUES
  ('manga-007', 45, 110, 30, 420),    -- Frieren
  ('manga-008', 50, 120, 35, 450),    -- Delicious in Dungeon
  ('manga-009', 60, 140, 40, 520),    -- Oshi no Ko
  ('manga-010', 55, 130, 38, 480);    -- Hell's Paradise

-- ============================================
-- 8. NOTIFICATIONS (For UDP broadcast testing)
-- ============================================

-- System notifications (admin-triggered)
INSERT INTO notifications (id, message) VALUES
  ('notif-001', '🔔 NEW CHAPTER: One Piece Chapter 1095 is now available!'),
  ('notif-002', '🎉 MAJOR UPDATE: Jujutsu Kaisen Season 2 anime adaptation announced!'),
  ('notif-003', '🚨 IMPORTANT: Chainsaw Man Part 2 begins next week!'),
  ('notif-004', '⭐ FEATURED: Kaguya-sama: Love is War completed with 22 volumes!'),
  ('notif-005', '🎮 COLLABORATION: Spy x Family mobile game launching next month!'),
  ('notif-006', '🏆 TOURNAMENT: Blue Lock anime wins Best Sports Anime 2024!'),
  ('notif-007', '✨ AWARD: Frieren: Beyond Journey''s End wins Manga of the Year!'),
  ('notif-008', '🔥 TRENDING: Delicious in Dungeon anime adaptation breaks streaming records!'),
  ('notif-009', '🎭 DRAMA: Oshi no Ko manga reaches 10 million copies sold!'),
  ('notif-010', '⚔️ BATTLE: Hell''s Paradise anime finale to air this weekend!');

-- ============================================
-- 9. ADMIN AUTOMATION SCRIPTS (For testing)
-- ============================================

-- Function to broadcast notification (UDP simulation)
CREATE OR REPLACE FUNCTION broadcast_notification(message TEXT) RETURNS VOID AS $$
BEGIN
  -- Insert into notifications table (UDP service will pick this up)
  INSERT INTO notifications (id, message) 
  VALUES ('notif-' || EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::TEXT, message);
  
  -- This would trigger the UDP service to broadcast in real implementation
  RAISE NOTICE 'UDP BROADCAST TRIGGERED: %', message;
END;
$$ LANGUAGE plpgsql;

-- Function to update manga stats (TCP simulation)
CREATE OR REPLACE FUNCTION update_manga_stats(
  p_manga_id TEXT,
  p_comment_delta INTEGER DEFAULT 0,
  p_like_delta INTEGER DEFAULT 0,
  p_chat_delta INTEGER DEFAULT 0
) RETURNS VOID AS $$
BEGIN
  INSERT INTO manga_stats (manga_id, comment_count, like_count, chat_count, weekly_score, updated_at)
  VALUES (
    p_manga_id,
    p_comment_delta,
    p_like_delta,
    p_chat_delta,
    (p_comment_delta + p_like_delta + p_chat_delta) * 10,
    CURRENT_TIMESTAMP
  )
  ON CONFLICT (manga_id) DO UPDATE SET
    comment_count = manga_stats.comment_count + p_comment_delta,
    like_count = manga_stats.like_count + p_like_delta,
    chat_count = manga_stats.chat_count + p_chat_delta,
    weekly_score = manga_stats.weekly_score + ((p_comment_delta + p_like_delta + p_chat_delta) * 10),
    updated_at = CURRENT_TIMESTAMP;
    
  RAISE NOTICE 'TCP STATS UPDATE: manga_id=%, comment_delta=%, like_delta=%, chat_delta=%', 
    p_manga_id, p_comment_delta, p_like_delta, p_chat_delta;
END;
$$ LANGUAGE plpgsql;

-- Function to simulate admin adding new manga
CREATE OR REPLACE FUNCTION add_new_manga(
  p_title TEXT,
  p_description TEXT,
  p_cover_url TEXT,
  p_status TEXT DEFAULT 'ongoing'
) RETURNS TEXT AS $$
DECLARE
  new_manga_id TEXT;
BEGIN
  new_manga_id := 'manga-' || (SELECT COUNT(*) + 1 FROM manga)::TEXT;
  
  INSERT INTO manga (id, title, description, cover_url, status)
  VALUES (new_manga_id, p_title, p_description, p_cover_url, p_status);
  
  -- Trigger activity feed entry
  INSERT INTO activity_feed (id, type, user_id, manga_id)
  VALUES ('activity-' || EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::TEXT, 'manga_update', 'admin-001', new_manga_id);
  
  -- Broadcast notification
  PERFORM broadcast_notification('📚 NEW MANGA: ' || p_title || ' has been added to the library!');
  
  RETURN new_manga_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 10. TEST SCENARIOS SETUP (Admin automation ready)
-- ============================================

-- Scenario 1: Create high-activity test environment
SELECT broadcast_notification('⚡ TEST SCENARIO: High activity simulation starting on One Piece!');
SELECT update_manga_stats('manga-001', 50, 120, 30);

-- Scenario 2: Simulate new manga release event
SELECT add_new_manga(
  'Demon Slayer: Kimetsu no Yaiba',
  'Tanjiro Kamado''s peaceful life is shattered when his family is slaughtered by demons. His sister Nezuko is the only survivor, but she has been transformed into a demon herself. Tanjiro sets out to become a demon slayer to avenge his family and cure his sister.',
  'https://example.com/covers/demonslayer.jpg',
  'completed'
);

-- Scenario 3: Simulate chat room explosion
SELECT broadcast_notification('💬 TEST ALERT: Chat room activity surge detected in Jujutsu Kaisen discussion!');
SELECT update_manga_stats('manga-002', 0, 0, 100);

-- Scenario 4: Weekly ranking reset simulation
SELECT broadcast_notification('🏆 WEEKLY RESET: Manga rankings have been refreshed! Check the new hot manga list.');
UPDATE manga_stats SET weekly_score = weekly_score * 0.7, updated_at = CURRENT_TIMESTAMP;

-- ============================================
-- END OF SEED DATA
-- ============================================

RAISE NOTICE '✅ DATABASE SEEDING COMPLETE! Ready for protocol testing:';
RAISE NOTICE '- HTTP API: Test CRUD operations on users, manga, comments';
RAISE NOTICE '- gRPC: Test full-text search on manga titles/descriptions';
RAISE NOTICE '- WebSocket: Connect to chat rooms for active manga';
RAISE NOTICE '- UDP: Monitor notification broadcasts from admin actions';
RAISE NOTICE '- TCP: Observe stats aggregation from activity events';
RAISE NOTICE '';
RAISE NOTICE 'Admin automation functions available:';
RAISE NOTICE '- broadcast_notification(message TEXT)';
RAISE NOTICE '- update_manga_stats(manga_id TEXT, comment_delta INT, like_delta INT, chat_delta INT)';
RAISE NOTICE '- add_new_manga(title TEXT, description TEXT, cover_url TEXT, status TEXT)';