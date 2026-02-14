#!/bin/bash

set -e

printf "Инициализация Forum Platform\n"
printf "==============================\n"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

if ! command -v docker &> /dev/null; then
    printf "${RED}Ошибка: Docker не установлен${NC}\n"
    exit 1
fi

printf "${GREEN}Docker найден${NC}\n"

# Создание конфигураций
if [ ! -f "./auth/.env" ]; then
    cat > ./auth/.env <<EOF
PSQL_USER=postgres
PSQL_PASS=postgres
PSQL_NAME=authdb
PSQL_HOST=postgres
PSQL_PORT=5432
JWT_SECRET=$(openssl rand -base64 32)
PORT=8012
SERVER_PORT=8012
EOF
    printf "${GREEN}Создан auth/.env${NC}\n"
fi

if [ ! -f "./forum/.env" ]; then
    cat > ./forum/.env <<EOF
MONGODB_URL=mongodb://admin:admin@mongodb:27017/forumdb?authSource=admin
DB_NAME=forumdb
EOF
    printf "${GREEN}Создан forum/.env${NC}\n"
fi

if [ ! -f "./gateway/.env" ]; then
    cat > ./gateway/.env <<EOF
AUTH_UPSTREAM=forum-auth:8012
FORUM_UPSTREAM=forum-backend:8000
JWT_SECRET=changeme
EOF
    printf "${GREEN}Создан gateway/.env${NC}\n"
fi

mkdir -p gateway/logs

printf "\nСборка и запуск сервисов...\n"
docker compose up --build -d

printf "\n${GREEN}Инициализация завершена${NC}\n"
