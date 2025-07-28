# SciFind 🧪

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker](https://img.shields.io/badge/docker-ready-blue.svg)](https://hub.docker.com/r/scifind/backend)
[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8.svg)](https://golang.org/dl/)

> **Next-generation scientific paper search and discovery platform** - Unified API for searching across multiple academic databases with AI-ready integration

## 🌟 Overview

SciFind Backend is a high-performance, production-ready scientific literature search engine that aggregates results from multiple academic databases into a single, unified API. Built with Go and designed for scale, it provides researchers, developers, and AI applications with instant access to millions of scientific papers through a single interface.

### Key Benefits
- **Unified Search**: One API call searches across ArXiv, Semantic Scholar, Exa, and Tavily
- **AI-Ready**: Native MCP (Model Context Protocol) support for seamless AI integration
- **Production-Grade**: Built for reliability with circuit breakers, caching, and monitoring
- **Cost-Optimized**: Intelligent rate limiting and caching to minimize API costs
- **Developer-Friendly**: Comprehensive documentation, SDKs, and Docker support

## 🚀 Quick Start

Get started in under 5 minutes:

```bash
# 1. Clone the repository
git clone https://github.com/scifind/backend.git
cd scifind-backend

# 2. Start with Docker Compose (includes all dependencies)
docker-compose up -d

# 3. Test the API
curl "http://localhost:8080/v1/search?query=quantum+computing&limit=5"
```

**That's it!** The API is now running at `http://localhost:8080` with full functionality.

For detailed setup instructions, see our [Quick Start Guide](docs/QUICKSTART.md).

## ✨ Features

### 🔍 Multi-Provider Search
- **ArXiv**: 2M+ papers in physics, mathematics, computer science
- **Semantic Scholar**: 200M+ papers with semantic understanding
- **Exa**: Neural search with web content understanding
- **Tavily**: Real-time web search with advanced filtering

### 🧠 AI Integration
- **MCP Protocol**: Native support for Model Context Protocol
- **Structured Responses**: JSON-LD formatted for AI consumption
- **Embeddings Ready**: Vector search capabilities (coming soon)
- **Chat Integration**: Direct integration with Claude, GPT, and other AI assistants

### ⚡ Performance & Reliability
- **Sub-second Response**: Parallel provider queries with intelligent caching
- **Circuit Breakers**: Graceful degradation when providers are down
- **Rate Limiting**: Provider-aware limits to prevent abuse
- **Auto-scaling**: Kubernetes-ready with horizontal pod autoscaling

### 📊 Analytics & Monitoring
- **Real-time Metrics**: Prometheus-compatible metrics
- **Usage Analytics**: Track search patterns and popular queries
- **Provider Health**: Monitor API health and response times
- **Cost Tracking**: Track API usage and costs per provider

### 🔐 Enterprise Features
- **API Key Management**: Secure key-based authentication
- **Rate Limiting**: Per-key and global rate limits
- **Audit Logging**: Complete audit trail for compliance
- **Multi-tenancy**: Support for multiple organizations

## 🏗️ Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| **Language** | Go 1.24+ | High-performance backend |
| **Framework** | Gin | Fast HTTP router |
| **Database** | PostgreSQL 15+ | Primary data storage |
| **Cache** | NATS KV | Distributed caching |
| **Messaging** | NATS.io | Event-driven architecture |
| **ORM** | GORM v2 | Database abstraction |
| **Container** | Docker | Containerization |
| **Orchestration** | Kubernetes | Production deployment |
| **Monitoring** | Prometheus + Grafana | Metrics and dashboards |
| **Testing** | Testcontainers | Integration testing |

## 📚 Documentation

We provide comprehensive documentation for all user types:

### 📖 [Quick Start Guide](docs/QUICKSTART.md)
Perfect for getting started quickly with Docker or local development.

### ⚙️ [Configuration Guide](docs/CONFIGURATION.md)
Detailed configuration options for production deployments.

### 🔌 [API Reference](docs/API_REFERENCE.md)
Complete API documentation with examples and response schemas.

### 🚀 [Deployment Guide](docs/DEPLOYMENT.md)
Production deployment strategies for Docker, Kubernetes, and cloud platforms.

### 🤖 [MCP Integration Guide](docs/MCP_INTEGRATION.md)
How to integrate SciFind with AI assistants using the Model Context Protocol.

## 🎯 Use Cases

### For Researchers
- **Literature Reviews**: Comprehensive search across all major databases
- **Stay Updated**: Automated alerts for new papers in your field
- **Citation Analysis**: Track paper impact and citations
- **Collaboration**: Find researchers working on similar topics

### For Developers
- **API Integration**: Simple REST API for scientific search
- **AI Applications**: Build AI-powered research assistants
- **Data Pipelines**: Automated paper collection and processing
- **Custom Frontends**: Build specialized research interfaces

### For Organizations
- **Research Portals**: Power institutional research platforms
- **Knowledge Management**: Centralized paper discovery
- **Competitive Intelligence**: Track research in specific domains
- **Compliance**: Ensure access to latest scientific literature

## 🔧 Configuration

SciFind is highly configurable through environment variables or YAML configuration:

```yaml
# Minimal configuration for development
server:
  port: 8080
  mode: "development"

database:
  sqlite:
    path: "./scifind.db"

# Add your API keys for enhanced functionality
providers:
  semantic_scholar:
    api_key: "your_key_here"
  exa:
    api_key: "your_key_here"
  tavily:
    api_key: "your_key_here"
```

See [Configuration Guide](docs/CONFIGURATION.md) for production-ready configurations.

## 🧪 Testing

Our comprehensive test suite ensures reliability:

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run integration tests (requires Docker)
make test-integration

# Run benchmarks
make benchmark
```

## 🤝 Contributing

We welcome contributions from the community! Here's how to get started:

### Quick Contribution Guide

1. **Fork & Clone**
   ```bash
   git clone https://github.com/your-username/scifind-backend.git
   cd scifind-backend
   ```

2. **Setup Development Environment**
   ```bash
   make dev-setup
   ```

3. **Create Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Make Changes & Test**
   ```bash
   make test
   make lint
   ```

5. **Submit Pull Request**
   - Ensure all tests pass
   - Add tests for new features
   - Update documentation
   - Follow our [Contributing Guidelines](CONTRIBUTING.md)

### Development Setup

```bash
# Install dependencies
go mod download

# Start development services
make dev-services

# Run in development mode
make dev
```

### Code Standards
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Add comprehensive tests (aim for >80% coverage)
- Document all public APIs
- Use conventional commit messages

## 🐛 Troubleshooting

### Common Issues

**Port already in use**
```bash
# Check what's using port 8080
lsof -i :8080
# Or use a different port
export SCIFIND_SERVER_PORT=8081
```

**Database connection issues**
```bash
# Reset database
make db-reset
# Or use SQLite for development
export SCIFIND_DATABASE_SQLITE_PATH="./dev.db"
```

**Provider API errors**
- Check API keys are set correctly
- Verify rate limits haven't been exceeded
- Check provider status pages

### Getting Help

- 📖 **Documentation**: Check our [docs](docs/) first
- 🐛 **Issues**: [Report bugs here](https://github.com/scifind/backend/issues)
- 💬 **Discussions**: [Join our community](https://github.com/scifind/backend/discussions)
- 📧 **Email**: support@scifind.org

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **ArXiv** for providing open access to scientific papers
- **Semantic Scholar** for their comprehensive academic API
- **Exa** for neural search capabilities
- **Tavily** for real-time web search
- **Go Community** for excellent tooling and libraries

## 🔗 Links

- **Website**: [https://scifind.org](https://scifind.org)
- **Documentation**: [https://docs.scifind.org](https://docs.scifind.org)
- **API Playground**: [https://api.scifind.org](https://api.scifind.org)
- **Docker Hub**: [https://hub.docker.com/r/scifind/backend](https://hub.docker.com/r/scifind/backend)
- **Discord Community**: [https://discord.gg/scifind](https://discord.gg/scifind)

---

<div align="center">
  <p>
    <strong>SciFind Backend</strong> - Empowering scientific discovery through unified search
  </p>
  <p>
    Built with ❤️ by the SciFind team and contributors
  </p>
</div>