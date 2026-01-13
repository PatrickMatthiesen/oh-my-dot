#!/usr/bin/env fish
# Docker Command Completion for Fish
# Enables docker shell completions and aliases

# Check if docker is installed
if command -v docker >/dev/null 2>&1
    # Common docker aliases
    alias d='docker'
    alias dps='docker ps'
    alias dpsa='docker ps -a'
    alias di='docker images'
    alias drm='docker rm'
    alias drmi='docker rmi'
    alias dexec='docker exec -it'
    alias dlog='docker logs'
    alias dlogf='docker logs -f'
    alias dstop='docker stop'
    alias dstart='docker start'
    alias drestart='docker restart'
    alias dbuild='docker build'
    alias dpull='docker pull'
    alias dpush='docker push'
    alias drun='docker run'
    
    # Docker Compose aliases
    if command -v docker-compose >/dev/null 2>&1; or docker compose version >/dev/null 2>&1
        alias dc='docker compose'
        alias dcup='docker compose up'
        alias dcupd='docker compose up -d'
        alias dcdown='docker compose down'
        alias dcps='docker compose ps'
        alias dclog='docker compose logs'
        alias dclogf='docker compose logs -f'
        alias dcexec='docker compose exec'
        alias dcbuild='docker compose build'
        alias dcrestart='docker compose restart'
    end
    
    # Docker cleanup aliases
    alias dprune='docker system prune -f'
    alias dprunea='docker system prune -af'
    alias dvprune='docker volume prune -f'
    alias dnprune='docker network prune -f'
end
