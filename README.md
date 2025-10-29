# 🛍️ Product Service

Product Service quản lý sản phẩm, categories, variants, và tags trong hệ thống ecommerce.

## 📋 Features

- ✅ Product CRUD operations
- ✅ Category management (with hierarchy)
- ✅ Product variants (size, color, etc.)
- ✅ Product images
- ✅ Tags/labels
- ✅ Search & filtering
- ✅ Pagination
- ✅ Stock management
- ✅ SEO fields
- ✅ JWT Authentication
- ✅ Soft delete

## 🗄️ Database Schema

### Tables:
- **categories** - Product categories with parent-child relationships
- **products** - Main products table
- **product_images** - Product images
- **product_variants** - Product variants (size, color options)
- **tags** - Product tags
- **product_tags** - Many-to-many relationship

## 📡 API Endpoints

### Products

#### Public Endpoints (No auth required)
```bash
GET    /products                    # List products with filters
GET    /products/:id                # Get product detail
```

#### Protected Endpoints (Auth required)
```bash
POST   /products                    # Create product
PUT    /products/:id                # Update product
DELETE /products/:id                # Delete product
```

### Categories

#### Public Endpoints
```bash
GET    /categories                  # List all categories
GET    /categories/root             # Get root categories
GET    /categories/:id              # Get category detail
```

#### Protected Endpoints
```bash
POST   /categories                  # Create category
PUT    /categories/:id              # Update category
DELETE /categories/:id              # Delete category
```

### Health Check
```bash
GET    /health                      # Service health status
```

## 🚀 Quick Start

### 1. With Docker Compose (Recommended)
```bash
# Start all services
docker-compose up -d

# Product Service will be available at:
# http://localhost:8082
```

### 2. Run Locally
```bash
# Install dependencies
cd product-service
go mod download

# Run migrations
psql -h localhost -U admin -d ecommerce -f migrations/001_initial_schema.sql

# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=admin
export DB_PASSWORD=admin
export DB_NAME=ecommerce
export JWT_SECRET=your-secret-key

# Run service
go run cmd/main.go
```

## 📝 Example Requests

### Create Product
```bash
curl -X POST http://localhost:8082/products \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "iPhone 15 Pro",
    "slug": "iphone-15-pro",
    "description": "Latest iPhone with advanced features",
    "short_description": "Apple iPhone 15 Pro - 256GB",
    "category_id": 2,
    "price": 999.99,
    "compare_at_price": 1199.99,
    "sku": "IPH15PRO256",
    "track_inventory": true,
    "stock_quantity": 50,
    "status": "active",
    "is_featured": true
  }'
```

### List Products with Filters
```bash
# All products
curl http://localhost:8082/products

# Filter by category
curl "http://localhost:8082/products?category_id=1"

# Search products
curl "http://localhost:8082/products?q=iphone"

# Filter by price range
curl "http://localhost:8082/products?min_price=500&max_price=1000"

# In stock only
curl "http://localhost:8082/products?in_stock=true"

# Sort by price
curl "http://localhost:8082/products?sort=price_asc"

# Pagination
curl "http://localhost:8082/products?page=1&page_size=20"

# Combine filters
curl "http://localhost:8082/products?category_id=1&in_stock=true&sort=price_asc&page=1"
```

### Get Product Detail
```bash
curl http://localhost:8082/products/1
```

### Update Product
```bash
curl -X PUT http://localhost:8082/products/1 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "price": 899.99,
    "stock_quantity": 45,
    "status": "active"
  }'
```

### Create Category
```bash
curl -X POST http://localhost:8082/categories \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Smartphones",
    "slug": "smartphones",
    "description": "Mobile phones and accessories",
    "parent_id": 1,
    "is_active": true,
    "display_order": 1
  }'
```

### List Categories
```bash
# All categories
curl http://localhost:8082/categories

# Active categories only
curl "http://localhost:8082/categories?active_only=true"

# Root categories
curl http://localhost:8082/categories/root
```

## 🔧 Configuration

Environment variables:

```bash
# Server
PORT=8082

# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=admin
DB_PASSWORD=admin
DB_NAME=ecommerce
DB_SSLMODE=disable

# JWT (same as user-service)
JWT_SECRET=production-secret-key-change-this

# Redis (optional)
REDIS_ADDR=redis:6379
REDIS_PASSWORD=
REDIS_DB=0

# External services
USER_SERVICE_URL=http://user-service:8081
```

## 📊 Database Models

### Product
```go
type Product struct {
    ID                uint
    Name              string
    Slug              string
    Description       string
    CategoryID        uint
    Price             float64
    SKU               string
    StockQuantity     int
    Status            string  // draft, active, archived
    IsFeatured        bool
    // ... more fields
}
```

### Category
```go
type Category struct {
    ID           uint
    Name         string
    Slug         string
    ParentID     *uint
    IsActive     bool
    DisplayOrder int
    // ... more fields
}
```

## 🧪 Testing

```bash
# Run tests
go test ./...

# With coverage
go test -cover ./...

# Test specific package
go test ./internal/application/...
```

## 🔐 Authentication

Product Service validates JWT tokens generated by User Service. The same `JWT_SECRET` must be used across both services.

**Public endpoints:** Can be accessed without authentication
**Protected endpoints:** Require valid JWT token in Authorization header

```bash
Authorization: Bearer <your_jwt_token>
```

## 📈 Performance

- Indexes on frequently queried fields (slug, category_id, price, status)
- Soft delete for data recovery
- Prepared statements for SQL queries
- Connection pooling
- Optional Redis caching (not implemented yet)

## 🚧 TODO / Future Enhancements

- [ ] Redis caching for products and categories
- [ ] Product reviews & ratings
- [ ] Product images upload handling
- [ ] Elasticsearch for better search
- [ ] Product variants API endpoints
- [ ] Bulk operations
- [ ] Export products to CSV
- [ ] Product analytics

## 📚 Dependencies

- Go 1.23+
- PostgreSQL 15+
- GORM (ORM)
- JWT for authentication
- go-playground/validator (validation)
- gosimple/slug (slug generation)

## 🐛 Troubleshooting

### Port already in use
```bash
# Check what's using port 8082
lsof -i :8082

# Kill the process
kill -9 <PID>
```

### Database connection issues
```bash
# Check if postgres is running
docker ps | grep postgres

# Check logs
docker logs postgres
```

### Migration not applied
```bash
# Manually run migration
docker exec -i postgres psql -U admin -d ecommerce < product-service/migrations/001_initial_schema.sql
```

## 📞 Support

For issues or questions:
- Check logs: `docker logs product-service`
- Database logs: `docker logs postgres`
- Health check: `curl http://localhost:8082/health`

## 📝 License

MIT