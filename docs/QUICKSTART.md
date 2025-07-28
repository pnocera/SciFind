# SciFind Backend Quick Start Guide

Get the SciFind backend up and running in under 5 minutes with this beginner-friendly guide.

## 🚀 Prerequisites

Before starting, ensure you have:

- **Go 1.24+** - [Download here](https://golang.org/dl/)
- **PostgreSQL 15+** (production) or **SQLite** (development)
- **NATS.io server 2.10+** - [Download here](https://nats.io/download/)
- **Git** - [Download here](https://git-scm.com/downloads)

## 📦 Quick Setup

### 1. Clone & Install

```bash
# Clone the repository
git clone <repository-url>
cd scifind-backend

# Install dependencies
go mod download
```

### 2. Quick Configuration

Create a minimal `config.yaml` file:

```yaml
# config.yaml
server:
  port: 8080
  host: "0.0.0.0"
  mode: "development"

database:
  sqlite:
    path: "./scifind.db"

nats:
  url: "nats://localhost:4222"
  cluster_id: "scifind-cluster"
  client_id: "scifind-backend"

providers:
  arxiv:
    enabled: true
    base_url: "http://export.arxiv.org/api/query"
    rate_limit: "3s"
  
  semantic_scholar:
    enabled: true
    api_key: "your_api_key_here"
```

### 3. Start Services

```bash
# Start NATS server
nats-server -js

# Start the application
go run cmd/server/main.go
```

### 4. Test Your Setup

```bash
# Health check
curl http://localhost:8080/health

# Search for papers
curl "http://localhost:8080/v1/search?query=machine+learning&limit=5"

# Get API documentation
curl http://localhost:8080/docs
```

## 🐳 Docker Quick Start

For an even faster setup:

```bash
# Build and run with Docker Compose
docker-compose up -d

# Test the API
curl "http://localhost:8080/v1/search?query=quantum+computing"
```

## ✅ Verification Checklist

- [ ] Server starts without errors
- [ ] Health endpoint returns 200 OK
- [ ] Search endpoint returns results
- [ ] Database connection successful
- [ ] NATS connection established

## 🆘 Troubleshooting

### Common Issues

**Port already in use:**
```bash
lsof -ti:8080 | xargs kill -9
```

**Database connection failed:**
- Check PostgreSQL is running: `pg_isready -h localhost`
- Verify connection string in config

**NATS connection failed:**
- Start NATS: `nats-server -js`
- Check NATS URL in config

**Missing API keys:**
- Get free API keys from:
  - [Semantic Scholar](https://www.semanticscholar.org/product/api)
  - [Exa](https://exa.ai/api)
  - [Tavily](https://tavily.com/)

## 🎯 Next Steps

1. **Configure providers** - Add your API keys to `config.yaml`
2. **Explore the API** - Visit `http://localhost:8080/docs`
3. **Run tests** - Execute `make test` to verify everything works
4. **Read the full documentation** - Check out the other docs in this folder

## 📞 Need Help?

- Check the [troubleshooting section](#troubleshooting)
- Review the [configuration guide](CONFIGURATION.md)
- Open an issue on GitHub with logs and configuration details