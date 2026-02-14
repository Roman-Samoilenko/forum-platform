# Forum Platform — Microservice Architecture

Платформа представляет собой распределенную систему для создания высоконагруженных форумов и сообществ, построенную на микросервисной архитектуре с единой точкой входа через API Gateway (OpenResty + Lua), который обеспечивает маршрутизацию, JWT-валидацию и защиту от несанкционированного доступа. Система состоит из трех независимых сервисов: Auth Service на Go с PostgreSQL для управления идентификацией, Forum Backend на FastAPI с MongoDB для работы с контентом и медиа, и Gateway для проксирования трафика, что обеспечивает горизонтальное масштабирование, отказоустойчивость и возможность независимого развертывания компонентов.

## Обзор архитектуры

Платформа реализует паттерн **API Gateway + Backend for Frontend (BFF)** с централизованной аутентификацией через JWT и полиглотной персистенцией данных (PostgreSQL для транзакционных операций, MongoDB для документно-ориентированного хранения).

### Архитектура высокого уровня

```mermaid
graph TB
    Client[Browser / API Client]
    
    subgraph "Edge Layer"
        GW[Gateway - OpenResty<br/>:80, :443<br/>Nginx + LuaJIT]
        Static[Static Assets<br/>public/, private/]
    end
    
    subgraph "Application Layer"
        AUTH[Auth Service<br/>:8080<br/>Go + net/http]
        FORUM[Forum Backend<br/>:8000<br/>FastAPI + Uvicorn]
    end
    
    subgraph "Data Layer"
        PG[(PostgreSQL<br/>:5432<br/>Users & Auth)]
        MG[(MongoDB<br/>:27017<br/>Topics & Media)]
    end
    
    Client -->|HTTP/HTTPS| GW
    GW -->|Serve| Static
    GW -->|POST /auth<br/>login/register| AUTH
    GW -->|GET/POST /api/*<br/>GET /forum<br/>JWT validated| FORUM
    
    AUTH -->|bcrypt + JWT| PG
    FORUM -->|Motor async| MG
    
    GW -.->|JWT validation<br/>lua-resty-jwt| GW
    AUTH -.->|Set-Cookie<br/>HTTP-only JWT| GW
    
    style GW fill:#4a90e2,stroke:#2171c9,color:#fff
    style AUTH fill:#e27a4a,stroke:#c96021,color:#fff
    style FORUM fill:#4ae290,stroke:#21c962,color:#fff
    style PG fill:#336791,stroke:#1a3f5c,color:#fff
    style MG fill:#13aa52,stroke:#0d7a3a,color:#fff
```

### Принципы проектирования

**Разделение ответственности (Separation of Concerns)**

- **Edge Layer (Gateway):** Маршрутизация, JWT-валидация, статическая отдача, SSL termination, rate limiting
- **Application Layer:** Бизнес-логика без знания о сетевой топологии
- **Data Layer:** Полиглотная персистенция с оптимизацией под use-case

**Безопасность (Security in Depth)**

- JWT токены с HMAC-SHA256 подписью и ротацией ключей
- bcrypt с cost factor 10 для паролей
- HTTP-only cookies для защиты от XSS
- Lua-based JWT декодирование на уровне gateway (без обращения к backend)
- Минимальная поверхность атаки через reverse proxy

**Асинхронность и производительность**

- Асинхронная обработка в Forum через FastAPI + Motor (asyncio)
- Горутины в Auth Service для параллельных DB операций
- LuaJIT в Gateway для sub-millisecond JWT validation
- Connection pooling в PostgreSQL (HikariCP-style)

## Технологический стек

### Backend Services

| Компонент | Технология | Версия | Роль |
|-----------|-----------|--------|------|
| **API Gateway** | OpenResty (Nginx + LuaJIT) | 1.21+ | Reverse proxy, JWT validation, static serving |
| **Auth Service** | Go + net/http | 1.25 | User registration, authentication, JWT issuance |
| **Forum Backend** | Python + FastAPI | 3.11 / FastAPI 0.109+ | Topics, tags, media, search, SSR |
| **JWT Library** | lua-resty-jwt | Latest | Stateless token validation in Lua |

### Data Layer

| СУБД | Роль | Особенности |
|------|------|-------------|
| **PostgreSQL 15** | Auth storage | ACID, unique indexes on login, timestamped records |
| **MongoDB 7** | Forum content | JSONB-like storage, GridFS for media, text indexes |

### Security & Protocols

| Протокол/Техника | Применение |
|-----------------|-----------|
| **JWT (HMAC-SHA256)** | Stateless authentication между Gateway и Backend |
| **bcrypt** | Password hashing с cost factor 10 |
| **HTTP-only Cookies** | Secure token delivery, XSS protection |
| **TLS 1.2+** | Encrypted transport (production) |

### Infrastructure

| Технология | Применение |
|------------|-----------|
| **Docker** | Контейнеризация всех сервисов |
| **Docker Compose** | Локальная оркестрация (dev environment) |
| **Nginx Upstream** | Load balancing и health checks |

## Компоненты системы

### Gateway (OpenResty) — Edge Layer

Высокопроизводительный reverse proxy с программируемой логикой маршрутизации на Lua.

**Ключевые возможности:**

- **JWT Validation:** Декодирование и верификация токенов на уровне Nginx без обращения к backend (<1ms latency)
- **Route-based Authorization:** Публичные маршруты (`/public/*`, `/auth`) vs защищённые (`/forum`, `/api/*`)
- **Static Serving:** Разделение публичной (auth.html, reg.js) и приватной статики
- **Header Injection:** Передача `X-User-Login` в backend после валидации JWT
- **Graceful Degradation:** Автоматический редирект на `/public/auth.html` при истечении токена

**Файлы:**

- `gateway/nginx.conf` — конфигурация upstream, locations, Lua hooks
- `gateway/jwt-auth.lua` — модуль валидации JWT с проверкой exp/iat
- `gateway/jwt.lua` — базовая библиотека декодирования токенов
- `gateway/static_public/` — статические ресурсы (auth.html, favicon.ico, reg.js)

**Архитектурные паттерны:**

- **Backend for Frontend (BFF):** Gateway адаптирует внутренние API под нужды фронтенда
- **Circuit Breaker:** Nginx upstream конфиг с `max_fails` и `fail_timeout`
- **API Composition:** Единый endpoint `/api` проксирует множественные backend routes

### Auth Service (Go) — Identity Provider

Микросервис аутентификации построен на стандартной библиотеке Go (net/http) с чистой слоистой архитектурой.

**Архитектура:**

```mermaid
graph TD
    subgraph "Auth Service Layers"
        HTTP[HTTP Handlers<br/>handlers/]
        MW[Middleware<br/>pkg/middleware/]
        SVC[Service Layer<br/>service/token.go]
        STORE[Storage Layer<br/>storage/]
    end
    
    Client --> HTTP
    HTTP --> MW
    MW --> SVC
    SVC --> STORE
    STORE --> DB[(PostgreSQL)]
    
    MW -.->|Logging| HTTP
    MW -.->|Recovery| HTTP
```

**Endpoints:**

- `POST /auth` — unified login/registration endpoint (auto-register if login doesn't exist)
- `GET /health` — health check для orchestration

**Технические детали:**

- **Password Hashing:** bcrypt.GenerateFromPassword с DefaultCost (10 rounds)
- **JWT Generation:** Claims с user_login, exp (1 hour), iat (issued at)
- **Database Abstraction:** Интерфейс `ManagerDB` для testability и future MySQL/SQLite support
- **Context Timeouts:** 5-секундные таймауты для всех DB операций
- **Structured Logging:** log/slog с JSON форматом для production

**Security Considerations:**

- Уникальный индекс на `login` предотвращает race conditions при регистрации
- bcrypt автоматически солит пароли (salt embedded в hash)
- JWT секрет читается из переменной окружения (не hardcoded)

### Forum Backend (FastAPI) — Content Management

Асинхронный backend на FastAPI с серверным рендерингом через Jinja2 Templates.

**Архитектура:**

```mermaid
graph TD
    subgraph "Forum Backend Structure"
        MAIN[main.py<br/>FastAPI App]
        ROUTES[routes/<br/>forum, tags, authors, graph]
        TMPL[templates/<br/>Jinja2 SSR]
        DB[database.py<br/>Motor async client]
        GRAPH[graph.py<br/>Related topics algorithm]
        MODELS[models.py<br/>Pydantic schemas]
    end
    
    MAIN --> ROUTES
    ROUTES --> TMPL
    ROUTES --> DB
    ROUTES --> GRAPH
    ROUTES --> MODELS
    DB --> MG[(MongoDB)]
```

**Функциональные модули:**

- **Topics Management:** CRUD операций для тем форума с автогенерацией slug
- **Tag System:** Иерархическая классификация контента с счётчиками использования
- **Media Upload:** Загрузка файлов с сохранением как MongoDB Binary (GridFS-style)
- **Graph Builder:** Алгоритм построения связей между темами по автору, тегам, keywords
- **Full-text Search:** MongoDB text indexes для поиска по содержимому

**Коллекции MongoDB:**

- `topics` — содержимое тем (title, content, author, tags, media, links)
- `users` — агрегированная статистика пользователей (topic_count, last_activity)
- `tags` — метаданные тегов (name, count, related_tags)

**Performance Optimizations:**

- Асинхронный драйвер Motor для non-blocking I/O
- Индексирование полей author, tags, created_at
- Pagination с limit/skip для больших списков
- Jinja2 template caching для повторяющихся страниц

## Потоки данных

### Authentication Flow (Login/Register)

```mermaid
sequenceDiagram
    participant C as Client (Browser)
    participant G as Gateway (Nginx)
    participant A as Auth Service (Go)
    participant PG as PostgreSQL
    
    C->>G: POST /auth<br/>{login, password}
    G->>A: Proxy request
    
    A->>PG: SELECT * FROM users WHERE login=?
    
    alt User exists
        PG-->>A: Return user row
        A->>A: bcrypt.CompareHashAndPassword
        alt Password valid
            A->>A: Generate JWT (exp: 1h)
            A-->>G: 200 OK + Set-Cookie (JWT, HttpOnly)
            G-->>C: Response + cookie
        else Password invalid
            A-->>G: 401 Unauthorized
            G-->>C: Invalid credentials
        end
    else User doesn't exist (Auto-register)
        A->>A: bcrypt.GenerateFromPassword
        A->>PG: INSERT INTO users (login, pass_hash)
        PG-->>A: User created
        A->>A: Generate JWT
        A-->>G: 201 Created + Set-Cookie
        G-->>C: Account created
    end
```

### Authorized Request Flow (Forum Access)

```mermaid
sequenceDiagram
    participant C as Client
    participant G as Gateway (Lua)
    participant F as Forum Backend
    participant MG as MongoDB
    
    C->>G: GET /forum<br/>Cookie: jwt=eyJ0...
    
    G->>G: access_by_lua_block:<br/>Decode JWT
    
    alt JWT valid (exp > now)
        G->>G: Extract user_login<br/>Set X-User-Login header
        G->>F: Proxy to FastAPI<br/>Headers: X-User-Login
        
        F->>MG: db.topics.find().sort(created_at: -1)
        MG-->>F: Topic list
        
        F->>F: Render Jinja2 template<br/>(forum_home.html)
        F-->>G: HTML response
        G-->>C: 200 OK + rendered page
    else JWT expired or invalid
        G-->>C: 302 Redirect to /public/auth.html
    end
```

### Topic Creation with Media Upload

```mermaid
sequenceDiagram
    participant C as Client
    participant G as Gateway
    participant F as Forum Backend
    participant MG as MongoDB
    
    C->>G: POST /api/topics<br/>multipart/form-data<br/>(title, content, files, tags)
    G->>G: Validate JWT
    G->>F: Proxy + X-User-Login
    
    F->>F: Validate Pydantic model
    
    loop For each uploaded file
        F->>F: Read file bytes
        F->>MG: Store as Binary<br/>db.media.insert_one({data, filename, content_type})
        MG-->>F: media_id
    end
    
    F->>MG: db.topics.insert_one({<br/>title, content, author,<br/>tags, media_ids, created_at<br/>})
    MG-->>F: topic_id
    
    F->>MG: db.users.update_one({login},<br/>{$inc: {topic_count: 1}})
    
    F-->>G: 201 Created {topic_id}
    G-->>C: Response
```

## Развертывание

### Топология (Docker Compose)

```mermaid
graph TB
    subgraph "Docker Network: forum_network"
        subgraph "Gateway Container"
            GW_NGINX[Nginx Process<br/>:80]
            GW_LUA[Lua Modules]
        end
        
        subgraph "Auth Container"
            AUTH_GO[Go Binary<br/>:8080]
        end
        
        subgraph "Forum Container"
            FORUM_PY[Uvicorn<br/>:8000]
        end
        
        subgraph "Database Containers"
            PG_CTR[PostgreSQL<br/>:5432]
            MG_CTR[MongoDB<br/>:27017]
        end
    end
    
    GW_NGINX --> AUTH_GO
    GW_NGINX --> FORUM_PY
    AUTH_GO --> PG_CTR
    FORUM_PY --> MG_CTR
    
    Host[Host Machine<br/>localhost:80] --> GW_NGINX
```

### Структура репозитория

```text
forum-platform/
├── auth/                       # Go Auth Service
│   ├── cmd/app/main.go        # Entry point
│   ├── internal/
│   │   ├── handlers/          # HTTP handlers (auth, health, register)
│   │   ├── service/           # JWT token generation
│   │   └── storage/           # PostgreSQL abstraction
│   ├── pkg/
│   │   ├── middleware/        # Logger, recovery
│   │   └── reply.go           # HTTP response helpers
│   ├── Dockerfile
│   ├── docker-compose.yaml
│   └── docs/README.md
│
├── forum/                      # FastAPI Forum Backend
│   ├── app/
│   │   ├── main.py            # FastAPI app factory
│   │   ├── routes/            # Forum, tags, authors, graph stats
│   │   ├── templates/         # Jinja2 HTML templates
│   │   ├── database.py        # Motor async MongoDB client
│   │   ├── graph.py           # Related topics algorithm
│   │   └── models.py          # Pydantic schemas
│   ├── run.py                 # Uvicorn launcher
│   ├── Dockerfile
│   ├── docker-compose.yaml
│   ├── requirements.txt
│   └── docs/
│       ├── README.md
│       └── *.png              # Screenshots
│
├── gateway/                    # OpenResty API Gateway
│   ├── nginx.conf             # Nginx + Lua configuration
│   ├── jwt-auth.lua           # JWT validation module
│   ├── jwt.lua                # JWT decode library
│   ├── static_public/         # Public assets (auth.html, reg.js)
│   ├── Dockerfile
│   ├── docker-compose.yaml
│   └── README.md
│
└── README.md                   # This file
```

## Конфигурация

### Переменные окружения

**Auth Service:**

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `postgres` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | Database username |
| `DB_PASSWORD` | `postgres` | Database password |
| `DB_NAME` | `authdb` | Database name |
| `JWT_SECRET` | `changeme` | HMAC signing key |
| `JWT_EXPIRY` | `3600` | Token lifetime in seconds (1 hour) |
| `PORT` | `8080` | HTTP server port |

**Forum Backend:**

| Variable | Default | Description |
|----------|---------|-------------|
| `MONGO_HOST` | `mongodb` | MongoDB host |
| `MONGO_PORT` | `27017` | MongoDB port |
| `MONGO_DB` | `forumdb` | Database name |
| `UVICORN_PORT` | `8000` | HTTP server port |

**Gateway:**

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_UPSTREAM` | `auth-service:8080` | Auth backend address |
| `FORUM_UPSTREAM` | `forum-backend:8000` | Forum backend address |
| `JWT_SECRET` | `changeme` | Same secret as Auth Service |

### Проверка работоспособности

```bash
# Health check Gateway
curl http://localhost/health

# Test Auth (register)
curl -X POST http://localhost/auth \
  -H "Content-Type: application/json" \
  -d '{"login":"testuser","password":"test123"}'

# Access Forum (will redirect if no JWT)
curl -L http://localhost/forum
```

## Безопасность

### Реализованные меры

- [x] JWT с HMAC-SHA256 и коротким TTL (1 час)
- [x] bcrypt для паролей (cost factor 10)
- [x] HTTP-only cookies (защита от XSS)
- [x] Уникальные индексы (защита от race conditions)
- [x] Context timeouts (защита от DoS)
- [x] Middleware recovery (устойчивость к паникам)

## Мониторинг и метрики

### Доступные эндпоинты

- `GET /health` (Auth Service) — Kubernetes liveness probe
- `GET /health` (Forum Backend) — FastAPI health check
- Nginx status page — `stub_status` модуль

### Логирование

- **Auth:** JSON structured logs (log/slog)
- **Forum:** Uvicorn access logs + Python logging
- **Gateway:** Nginx access.log + error.log

## Лицензия

MIT License
