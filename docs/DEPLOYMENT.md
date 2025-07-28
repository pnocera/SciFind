# SciFind Backend Deployment Guide

Comprehensive production deployment instructions and best practices for the SciFind backend.

## 📋 Table of Contents
- [Prerequisites](#prerequisites)
- [Deployment Options](#deployment-options)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Cloud Deployment](#cloud-deployment)
- [Environment Configuration](#environment-configuration)
- [Security Best Practices](#security-best-practices)
- [Monitoring & Observability](#monitoring--observability)
- [Scaling & Performance](#scaling--performance)
- [Backup & Disaster Recovery](#backup--disaster-recovery)
- [CI/CD Pipeline](#cicd-pipeline)
- [Troubleshooting](#troubleshooting)

## ✅ Prerequisites

Before deploying to production, ensure you have:

- **Production-ready configuration** (see [CONFIGURATION.md](CONFIGURATION.md))
- **SSL/TLS certificates** for HTTPS
- **Domain name** and DNS configured
- **Container registry** access (Docker Hub, AWS ECR, etc.)
- **Cloud provider account** (AWS, GCP, Azure, etc.)
- **Monitoring tools** (Prometheus, Grafana, etc.)
- **Backup strategy** in place

## 🚀 Deployment Options

### Option 1: Docker Compose (Recommended for small deployments)
### Option 2: Kubernetes (Recommended for production)
### Option 3: Cloud-native services (AWS ECS, GCP Cloud Run, etc.)

## 🐳 Docker Deployment

### 1. Build Production Image

```bash
# Build optimized production image
docker build -t scifind-backend:latest .

# Build with specific tag
docker build -t your-registry/scifind-backend:v1.0.0 .
```

### 2. Production Docker Compose

Create `docker-compose.prod.yml`:

```yaml
version: '3.8'

services:
  scifind-backend:
    image: scifind-backend:latest
    container_name: scifind-backend
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - SCIFIND_SERVER_MODE=production
      - SCIFIND_DATABASE_TYPE=postgres
      - SCIFIND_DATABASE_POSTGRESQL_DSN=postgres://user:pass@postgres:5432/scifind?sslmode=require
      - SCIFIND_NATS_URL=nats://nats:4222
      - SCIFIND_PROVIDERS_SEMANTIC_SCHOLAR_API_KEY=${SEMANTIC_SCHOLAR_API_KEY}
    depends_on:
      - postgres
      - nats
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  postgres:
    image: postgres:15-alpine
    container_name: scifind-postgres
    restart: unless-stopped
    environment:
      - POSTGRES_DB=scifind
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=secure_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  nats:
    image: nats:2.10-alpine
    container_name: scifind-nats
    restart: unless-stopped
    ports:
      - "4222:4222"
    volumes:
      - nats_data:/data

  nginx:
    image: nginx:alpine
    container_name: scifind-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - scifind-backend

volumes:
  postgres_data:
  nats_data:
```

### 3. Production Nginx Configuration

Create `nginx.conf`:

```nginx
events {
    worker_connections 1024;
}

http {
    upstream scifind_backend {
        server scifind-backend:8080;
    }

    # Redirect HTTP to HTTPS
    server {
        listen 80;
        server_name yourdomain.com;
        return 301 https://$server_name$request_uri;
    }

    # HTTPS Configuration
    server {
        listen 443 ssl http2;
        server_name yourdomain.com;

        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
        ssl_prefer_server_ciphers off;

        # Security headers
        add_header X-Frame-Options "SAMEORIGIN" always;
        add_header X-Content-Type-Options "nosniff" always;
        add_header X-XSS-Protection "1; mode=block" always;
        add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

        # Gzip compression
        gzip on;
        gzip_vary on;
        gzip_min_length 1024;
        gzip_types text/plain text/css application/json application/javascript text/xml application/xml;

        # Rate limiting
        limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
        limit_req zone=api burst=20 nodelay;

        location / {
            proxy_pass http://scifind_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_connect_timeout 30s;
            proxy_send_timeout 30s;
            proxy_read_timeout 30s;
        }

        location /health {
            access_log off;
            proxy_pass http://scifind_backend/health;
        }
    }
}
```

### 4. Deploy with Docker Compose

```bash
# Create production environment file
cat > .env.prod << EOF
POSTGRES_PASSWORD=secure_password_$(openssl rand -hex 32)
SEMANTIC_SCHOLAR_API_KEY=your_api_key
EXA_API_KEY=your_api_key
TAVILY_API_KEY=your_api_key
EOF

# Deploy with Docker Compose
docker-compose -f docker-compose.prod.yml up -d

# Check deployment
docker-compose -f docker-compose.prod.yml ps
```

## ☸️ Kubernetes Deployment

### 1. Kubernetes Manifests

Create `k8s/` directory with the following files:

#### Namespace
```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: scifind
  labels:
    name: scifind
```

#### ConfigMap
```yaml
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: scifind-config
  namespace: scifind
data:
  config.yaml: |
    server:
      port: 8080
      host: "0.0.0.0"
      mode: "production"
    database:
      type: "postgres"
      postgresql:
        dsn: "postgres://user:password@postgres-service:5432/scifind?sslmode=require"
    nats:
      url: "nats://nats-service:4222"
    providers:
      semantic_scholar:
        enabled: true
        api_key: "${SEMANTIC_SCHOLAR_API_KEY}"
```

#### Secrets
```yaml
# k8s/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: scifind-secrets
  namespace: scifind
type: Opaque
stringData:
  semantic-scholar-api-key: "your_api_key_here"
  exa-api-key: "your_api_key_here"
  tavily-api-key: "your_api_key_here"
```

#### Deployment
```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: scifind-backend
  namespace: scifind
  labels:
    app: scifind-backend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: scifind-backend
  template:
    metadata:
      labels:
        app: scifind-backend
    spec:
      containers:
      - name: scifind-backend
        image: your-registry/scifind-backend:v1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: SCIFIND_SERVER_MODE
          value: "production"
        - name: SCIFIND_DATABASE_POSTGRESQL_DSN
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: dsn
        - name: SEMANTIC_SCHOLAR_API_KEY
          valueFrom:
            secretKeyRef:
              name: scifind-secrets
              key: semantic-scholar-api-key
        volumeMounts:
        - name: config
          mountPath: /app/config
          readOnly: true
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: scifind-config
```

#### Service
```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: scifind-backend-service
  namespace: scifind
spec:
  selector:
    app: scifind-backend
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP
```

#### Ingress
```yaml
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: scifind-ingress
  namespace: scifind
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - yourdomain.com
    secretName: scifind-tls
  rules:
  - host: yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: scifind-backend-service
            port:
              number: 80
```

### 2. Deploy to Kubernetes

```bash
# Apply all manifests
kubectl apply -f k8s/

# Check deployment status
kubectl get pods -n scifind
kubectl get services -n scifind
kubectl get ingress -n scifind

# Scale deployment
kubectl scale deployment scifind-backend --replicas=5 -n scifind
```

## ☁️ Cloud Deployment

### AWS ECS with Fargate

#### 1. Create ECS Task Definition
```json
{
  "family": "scifind-backend",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole",
  "containerDefinitions": [
    {
      "name": "scifind-backend",
      "image": "your-account.dkr.ecr.region.amazonaws.com/scifind-backend:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {"name": "SCIFIND_SERVER_MODE", "value": "production"},
        {"name": "SCIFIND_DATABASE_TYPE", "value": "postgres"},
        {"name": "SCIFIND_DATABASE_POSTGRESQL_DSN", "value": "postgres://user:pass@rds-endpoint:5432/scifind"}
      ],
      "secrets": [
        {
          "name": "SEMANTIC_SCHOLAR_API_KEY",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:scifind-api-keys:semantic_scholar"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/scifind-backend",
          "awslogs-region": "us-west-2",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

#### 2. Deploy with AWS CLI
```bash
# Create ECS cluster
aws ecs create-cluster --cluster-name scifind-cluster

# Register task definition
aws ecs register-task-definition --cli-input-json file://task-definition.json

# Create service
aws ecs create-service \
  --cluster scifind-cluster \
  --service-name scifind-backend \
  --task-definition scifind-backend:1 \
  --desired-count 3 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345],assignPublicIp=ENABLED}"
```

### Google Cloud Run

#### 1. Deploy to Cloud Run
```bash
# Build and push to Google Container Registry
gcloud builds submit --tag gcr.io/your-project/scifind-backend

# Deploy to Cloud Run
gcloud run deploy scifind-backend \
  --image gcr.io/your-project/scifind-backend \
  --platform managed \
  --region us-central1 \
  --memory 1Gi \
  --cpu 1 \
  --max-instances 10 \
  --set-env-vars "SCIFIND_SERVER_MODE=production,SCIFIND_DATABASE_TYPE=postgres"
```

### Azure Container Instances

#### 1. Deploy with Azure CLI
```bash
# Create resource group
az group create --name scifind-rg --location eastus

# Create container instance
az container create \
  --resource-group scifind-rg \
  --name scifind-backend \
  --image your-registry.azurecr.io/scifind-backend:latest \
  --cpu 1 \
  --memory 1 \
  --ports 8080 \
  --environment-variables SCIFIND_SERVER_MODE=production
```

## 🔧 Environment Configuration

### Production Environment Variables

```bash
# Server
export SCIFIND_SERVER_MODE=production
export SCIFIND_SERVER_PORT=8080
export SCIFIND_SERVER_HOST=0.0.0.0

# Database
export SCIFIND_DATABASE_TYPE=postgres
export SCIFIND_DATABASE_POSTGRESQL_DSN="postgres://user:pass@prod-db:5432/scifind?sslmode=require"

# NATS
export SCIFIND_NATS_URL="nats://prod-nats:4222"
export SCIFIND_NATS_CLUSTER_ID="scifind-cluster"

# Security
export SCIFIND_SECURITY_API_KEYS="prod-key-1,prod-key-2"

# Monitoring
export SCIFIND_MONITORING_ENABLED=true
export SCIFIND_MONITORING_METRICS_PORT=9090
```

### Environment-Specific Configurations

#### Development
```yaml
server:
  mode: "debug"
  port: 8080

database:
  type: "sqlite"
  sqlite:
    path: "./dev.db"

logging:
  level: "debug"
```

#### Staging
```yaml
server:
  mode: "release"
  port: 8080

database:
  type: "postgres"
  postgresql:
    dsn: "postgres://user:pass@staging-db:5432/scifind?sslmode=require"

logging:
  level: "info"
```

#### Production
```yaml
server:
  mode: "release"
  port: 8080

database:
  type: "postgres"
  postgresql:
    dsn: "postgres://user:pass@prod-db:5432/scifind?sslmode=require"
    max_connections: 50

logging:
  level: "warn"
```

## 🔒 Security Best Practices

### 1. SSL/TLS Configuration
```bash
# Generate SSL certificates
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout ssl/private.key -out ssl/certificate.crt
```

### 2. Environment Secrets
```bash
# Use environment variables for secrets
export SCIFIND_PROVIDERS_SEMANTIC_SCHOLAR_API_KEY="your_secure_key"
export SCIFIND_DATABASE_PASSWORD="secure_password"
```

### 3. Network Security
- Use VPC/private networks
- Configure security groups/firewalls
- Enable TLS for all communications
- Use secrets management (AWS Secrets Manager, Azure Key Vault)

### 4. Container Security
```dockerfile
# Use distroless base image
FROM gcr.io/distroless/static:nonroot
USER nonroot:nonroot
```

### 5. Kubernetes Security
```yaml
# Security context
securityContext:
  runAsNonRoot: true
  runAsUser: 65534
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
    - ALL
```

## 📊 Monitoring & Observability

### 1. Prometheus Configuration
```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'scifind-backend'
    static_configs:
      - targets: ['localhost:9090']
```

### 2. Grafana Dashboard
Import the dashboard from `docs/grafana/scifind-dashboard.json`

### 3. Alerting Rules
```yaml
# alert-rules.yml
groups:
  - name: scifind-alerts
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
        for: 5m
        annotations:
          summary: "High error rate detected"
```

## 📈 Scaling & Performance

### Horizontal Scaling
```yaml
# Kubernetes HPA
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: scifind-backend-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: scifind-backend
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### Database Scaling
```sql
-- PostgreSQL connection pooling
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
```

### Load Balancing
```yaml
# AWS ALB configuration
LoadBalancer:
  Type: AWS::ElasticLoadBalancingV2::LoadBalancer
  Properties:
    Type: application
    Scheme: internet-facing
    SecurityGroups:
      - !Ref LoadBalancerSecurityGroup
    Subnets:
      - !Ref PublicSubnet1
      - !Ref PublicSubnet2
```

## 💾 Backup & Disaster Recovery

### 1. Database Backup
```bash
# PostgreSQL backup
pg_dump -h localhost -U user -d scifind > backup_$(date +%Y%m%d_%H%M%S).sql

# Automated backup script
#!/bin/bash
BACKUP_DIR="/backups/postgres"
DATE=$(date +%Y%m%d_%H%M%S)
pg_dump -h postgres-service -U user -d scifind | gzip > $BACKUP_DIR/backup_$DATE.sql.gz
```

### 2. Kubernetes Backup
```bash
# Backup persistent volumes
kubectl get pv -o yaml > pv-backup.yaml
kubectl get pvc -o yaml > pvc-backup.yaml

# Backup secrets
kubectl get secrets -n scifind -o yaml > secrets-backup.yaml
```

### 3. Disaster Recovery Plan
```yaml
# DR checklist
- Database: Point-in-time recovery (PITR)
- Application: Blue-green deployment
- Data: Cross-region replication
- DNS: Health check failover
```

## 🔄 CI/CD Pipeline

### GitHub Actions
```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]
  workflow_dispatch:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.24'
      - run: make test

  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build and push
        uses: docker/build-push-action@v3
        with:
          push: true
          tags: your-registry/scifind-backend:${{ github.sha }}

  deploy:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to production
        run: |
          kubectl set image deployment/scifind-backend scifind-backend=your-registry/scifind-backend:${{ github.sha }} -n scifind
```

### GitLab CI/CD
```yaml
# .gitlab-ci.yml
stages:
  - test
  - build
  - deploy

test:
  stage: test
  script:
    - make test

build:
  stage: build
  script:
    - docker build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA

deploy:
  stage: deploy
  script:
    - kubectl set image deployment/scifind-backend scifind-backend=$CI_REGISTRY_IMAGE:$CI_COMMIT_SHA -n scifind
```

## 🆘 Troubleshooting

### Common Deployment Issues

#### 1. Container Won't Start
```bash
# Check logs
docker logs scifind-backend

# Check configuration
docker exec scifind-backend cat /app/config/config.yaml

# Check environment variables
docker exec scifind-backend env | grep SCIFIND
```

#### 2. Database Connection Issues
```bash
# Test database connection
docker exec scifind-backend pg_isready -h postgres -p 5432

# Check database logs
docker logs postgres
```

#### 3. SSL Certificate Issues
```bash
# Check certificate validity
openssl x509 -in ssl/certificate.crt -text -noout

# Test SSL connection
curl -I https://yourdomain.com
```

#### 4. Performance Issues
```bash
# Check resource usage
kubectl top pods -n scifind

# Check logs for errors
kubectl logs -f deployment/scifind-backend -n scifind
```

### Debug Commands
```bash
# Container debugging
kubectl exec -it deployment/scifind-backend -n scifind -- /bin/sh

# Network debugging
kubectl run debug-pod --image=busybox --rm -it -- /bin/sh

# DNS debugging
kubectl exec -it deployment/scifind-backend -n scifind -- nslookup postgres-service
```

## 📞 Support & Maintenance

### Regular Maintenance Tasks
- [ ] Update dependencies monthly
- [ ] Rotate API keys quarterly
- [ ] Review and update security patches
- [ ] Monitor performance metrics
- [ ] Test backup and recovery procedures
- [ ] Review and update documentation

### Monitoring Checklist
- [ ] Application health checks
- [ ] Database performance
- [ ] Provider API availability
- [ ] SSL certificate expiration
- [ ] Security scan results
- [ ] Backup verification

### Emergency Contacts
- **On-call rotation**: Set up PagerDuty/Opsgenie
- **Escalation procedures**: Document in runbook
- **Communication channels**: Slack/Teams alerts
- **Incident response**: Playbook available at `/docs/incident-response.md`