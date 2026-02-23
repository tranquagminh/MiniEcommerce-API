-- Seed Products (categories already inserted from first run)
INSERT INTO products (name, slug, description, short_description, category_id, price, compare_at_price, cost_per_item, sku, barcode, track_inventory, stock_quantity, low_stock_threshold, status, is_featured, meta_title, meta_description, created_at, updated_at) VALUES
-- Tai nghe & Audio (category_id = 3)
('Tai nghe Sony WH-1000XM5', 'tai-nghe-sony-wh-1000xm5', 'Tai nghe chống ồn cao cấp với chất lượng âm thanh tuyệt vời', 'Tai nghe Sony WH-1000XM5 chống ồn', 3, 7990000, 9990000, 6000000, 'SONY-WH1000XM5', '4548736132351', true, 50, 10, 'active', true, 'Tai nghe Sony WH-1000XM5', 'Mua tai nghe Sony WH-1000XM5 chính hãng', NOW(), NOW()),

('Loa JBL Charge 5', 'loa-jbl-charge-5', 'Loa bluetooth chống nước với âm bass mạnh mẽ', 'Loa JBL Charge 5 chống nước', 3, 4290000, 4990000, 3200000, 'JBL-CHARGE5', '6925281982002', true, 30, 5, 'active', false, 'Loa JBL Charge 5', 'Mua loa JBL Charge 5 chính hãng', NOW(), NOW()),

('AirPods Pro 2', 'airpods-pro-2', 'Tai nghe không dây Apple với chống ồn chủ động', 'Apple AirPods Pro thế hệ 2', 3, 6490000, 7490000, 5000000, 'APPLE-APP2', '194253397052', true, 100, 20, 'active', true, 'AirPods Pro 2', 'Mua AirPods Pro 2 chính hãng', NOW(), NOW()),

-- Laptop & Máy tính (category_id = 1)
('Laptop Dell XPS 13', 'laptop-dell-xps-13', 'Laptop cao cấp với màn hình InfinityEdge tuyệt đẹp', 'Dell XPS 13 Intel Core i7', 1, 32990000, 36990000, 28000000, 'DELL-XPS13', '884116417538', true, 20, 5, 'active', true, 'Laptop Dell XPS 13', 'Mua laptop Dell XPS 13 chính hãng', NOW(), NOW()),

('iPad Pro 12.9" M2', 'ipad-pro-12-9-m2', 'iPad Pro với chip M2 mạnh mẽ', 'Apple iPad Pro 12.9 inch chip M2', 1, 29990000, 32990000, 25000000, 'APPLE-IPADPRO', '194253265207', true, 25, 5, 'active', false, 'iPad Pro 12.9 M2', 'Mua iPad Pro M2 chính hãng', NOW(), NOW()),

-- Điện thoại (category_id = 2)
('iPhone 15 Pro Max', 'iphone-15-pro-max', 'iPhone mới nhất với camera 48MP và chip A17 Pro', 'Apple iPhone 15 Pro Max 256GB', 2, 34990000, 37990000, 30000000, 'APPLE-IP15PM', '194253402404', true, 80, 15, 'active', true, 'iPhone 15 Pro Max', 'Mua iPhone 15 Pro Max chính hãng', NOW(), NOW()),

-- Smart Watch (category_id = 4)
('Apple Watch Series 9', 'apple-watch-series-9', 'Đồng hồ thông minh với chip S9 mới', 'Apple Watch Series 9 GPS 45mm', 4, 10990000, NULL, 8500000, 'APPLE-AWS9', '194253478492', true, 40, 10, 'active', false, 'Apple Watch Series 9', 'Mua Apple Watch Series 9 chính hãng', NOW(), NOW()),

-- Camera (category_id = 5)
('Canon EOS R6 Mark II', 'canon-eos-r6-mark-ii', 'Máy ảnh mirrorless full-frame chuyên nghiệp', 'Canon EOS R6 Mark II Body', 5, 68990000, NULL, 55000000, 'CANON-R6M2', '4549292195064', true, 10, 2, 'active', false, 'Canon EOS R6 Mark II', 'Mua Canon EOS R6 Mark II chính hãng', NOW(), NOW()),

-- Gaming (category_id = 6)
('Bàn phím Gaming Razer', 'ban-phim-gaming-razer', 'Bàn phím cơ gaming với đèn RGB', 'Razer BlackWidow V4 Pro', 6, 3490000, NULL, 2500000, 'RAZER-BWV4', '8886419333784', true, 45, 10, 'active', false, 'Bàn phím Razer BlackWidow', 'Mua bàn phím Razer chính hãng', NOW(), NOW()),

('LG UltraGear Monitor 27"', 'lg-ultragear-monitor-27', 'Màn hình gaming 27 inch 144Hz', 'LG UltraGear 27GP850-B', 6, 8990000, NULL, 7000000, 'LG-27GP850', '8806091547392', true, 0, 5, 'active', false, 'Màn hình LG UltraGear 27"', 'Mua màn hình LG gaming chính hãng', NOW(), NOW());

-- Seed Tags
INSERT INTO tags (name, slug, created_at) VALUES
('Bán chạy', 'ban-chay', NOW()),
('Mới', 'moi', NOW()),
('Giảm giá', 'giam-gia', NOW()),
('Hot', 'hot', NOW());

-- Link products with tags (product_model_id, tag_model_id)
INSERT INTO product_tags (product_model_id, tag_model_id) VALUES
(1, 1), -- Sony headphone - Bán chạy
(2, 3), -- JBL - Giảm giá
(3, 1), -- AirPods - Bán chạy
(4, 4), -- Dell XPS - Hot
(6, 2), -- iPhone - Mới
(9, 3); -- Razer keyboard - Giảm giá
