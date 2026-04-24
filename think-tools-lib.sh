#!/bin/bash

set -euo pipefail

CONFIG_FILE="${HOME}/.think-tools.json"

load_config() {
    if [[ -f "$CONFIG_FILE" ]]; then
        GROQ_API_KEY="${GROQ_API_KEY:-$(jq -r '.api_key // empty' "$CONFIG_FILE" 2>/dev/null || echo "")}"
        GROQ_MODEL="${GROQ_MODEL:-$(jq -r '.model // "llama-3.3-70b-versatile"' "$CONFIG_FILE" 2>/dev/null || echo "llama-3.3-70b-versatile")}"
    fi
    
    GROQ_API_KEY="${GROQ_API_KEY:-}"
    GROQ_MODEL="${GROQ_MODEL:-llama-3.3-70b-versatile}"
}

call_groq() {
    local model="$1"
    local system_prompt="$2"
    local user_prompt="$3"
    local max_retries="${4:-2}"
    
    local payload
    payload=$(jq -n \
        --arg model "$model" \
        --arg system "$system_prompt" \
        --arg user "$user_prompt" \
        '{"model": $model, "messages": [{"role": "system", "content": $system}, {"role": "user", "content": $user}]}')
    
    local attempt=0
    local response
    
    while [[ $attempt -lt $max_retries ]]; do
        attempt=$((attempt + 1))
        
        response=$(curl -s -X POST "https://api.groq.com/openai/v1/chat/completions" \
            -H "Authorization: Bearer $GROQ_API_KEY" \
            -H "Content-Type: application/json" \
            -d "$payload")
        
        if echo "$response" | jq -e '.choices[0].message.content' >/dev/null 2>&1; then
            echo "$response"
            return 0
        fi
        
        if [[ $attempt -lt $max_retries ]]; then
            sleep 1
        fi
    done
    
    echo "$response" >&2
    return 1
}

stream_response() {
    local model="$1"
    local system_prompt="$2"
    local user_prompt="$3"
    local max_retries="${4:-2}"
    
    local payload
    payload=$(jq -n \
        --arg model "$model" \
        --arg system "$system_prompt" \
        --arg user "$user_prompt" \
        '{"model": $model, "messages": [{"role": "system", "content": $system}, {"role": "user", "content": $user}], "stream": true}')
    
    local attempt=0
    
    while [[ $attempt -lt $max_retries ]]; do
        attempt=$((attempt + 1))
        
        curl -s -X POST "https://api.groq.com/openai/v1/chat/completions" \
            -H "Authorization: Bearer $GROQ_API_KEY" \
            -H "Content-Type: application/json" \
            -d "$payload" \
            --no-buffer \
            | while IFS= read -r line; do
                if [[ "$line" =~ ^data:\ (.+) ]]; then
                    local data="${BASH_REMATCH[1]}"
                    if [[ "$data" == "[DONE]" ]]; then
                        break
                    fi
                    echo "$data" | jq -r '.choices[0].delta.content // empty'
                fi
            done
        
        if [[ ${PIPESTATUS[0]} -eq 0 ]]; then
            return 0
        fi
        
        if [[ $attempt -lt $max_retries ]]; then
            sleep 1
        fi
    done
    
    return 1
}

retry_on_failure() {
    local fn="$1"
    shift
    local max_retries="${1:-2}"
    shift
    
    local attempt=0
    local result
    
    while [[ $attempt -lt $max_retries ]]; do
        attempt=$((attempt + 1))
        
        if "$fn" "$@"; then
            return 0
        fi
        
        if [[ $attempt -lt $max_retries ]]; then
            sleep 1
        fi
    done
    
    return 1
}