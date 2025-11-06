#!/bin/bash
# Fluxbase Key Generator
# Generates JWT secret, service keys, or anon keys

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Banner
echo -e "${BLUE}"
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║              Fluxbase Key Generator                          ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check dependencies
if ! command -v openssl &> /dev/null; then
    echo -e "${RED}Error: openssl is required but not installed${NC}"
    exit 1
fi

# Prompt for key type
echo -e "${BLUE}Select key type to generate:${NC}"
echo ""
echo "1) JWT Secret    - Master secret for signing JWT tokens"
echo "                   (Required for Fluxbase to generate tokens)"
echo ""
echo "2) Service Key   - For backend services, cron jobs, admin scripts"
echo "                   (Bypasses RLS, full database access)"
echo ""
echo "3) Anon Key      - For client-side anonymous access"
echo "                   (JWT token with 'anon' role, respects RLS)"
echo ""
read -p "Enter choice [1-3]: " KEY_TYPE

case $KEY_TYPE in
    1)
        # ============================================================
        # JWT SECRET GENERATION
        # ============================================================

        echo ""
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${BLUE}           JWT Secret Generation${NC}"
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo ""

        echo -e "${YELLOW}The JWT secret is used to sign all JWT tokens (access & refresh tokens).${NC}"
        echo -e "${YELLOW}This is a critical secret - keep it secure!${NC}"
        echo ""

        # Generate secure random secret (256 bits = 32 bytes)
        JWT_SECRET=$(openssl rand -base64 32 | tr -d '\n')

        echo -e "${GREEN}✓ JWT secret generated${NC}"
        echo ""

        # Display the secret
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}                  YOUR JWT SECRET${NC}"
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        echo ""
        echo -e "${YELLOW}⚠️  SAVE THIS SECRET NOW - IT WILL ONLY BE SHOWN ONCE ⚠️${NC}"
        echo ""
        echo -e "${BLUE}JWT Secret:${NC}"
        echo -e "${GREEN}${JWT_SECRET}${NC}"
        echo ""
        echo -e "${BLUE}Length:${NC} $(echo -n "$JWT_SECRET" | wc -c) characters"
        echo ""
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        echo ""

        # Usage instructions
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${BLUE}           Configuration Instructions${NC}"
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo ""

        echo -e "${BLUE}1. Environment Variable:${NC}"
        cat <<'EOF'
export FLUXBASE_JWT_SECRET="<your-jwt-secret>"
EOF
        echo ""

        echo -e "${BLUE}2. Configuration File (fluxbase.yaml):${NC}"
        cat <<'EOF'
auth:
  jwt_secret: "<your-jwt-secret>"
EOF
        echo ""

        echo -e "${BLUE}3. Docker Compose:${NC}"
        cat <<'EOF'
environment:
  - FLUXBASE_JWT_SECRET=<your-jwt-secret>
EOF
        echo ""

        echo -e "${BLUE}4. Kubernetes Secret:${NC}"
        cat <<EOF
kubectl create secret generic fluxbase-jwt-secret \\
  --from-literal=jwt-secret="${JWT_SECRET}"
EOF
        echo ""

        echo -e "${YELLOW}⚠️  CRITICAL SECURITY WARNINGS:${NC}"
        echo -e "${YELLOW}- Never commit the JWT secret to version control${NC}"
        echo -e "${YELLOW}- Store in a secrets manager (Vault, AWS Secrets Manager, etc.)${NC}"
        echo -e "${YELLOW}- If compromised, ALL issued JWT tokens are invalid${NC}"
        echo -e "${YELLOW}- Changing this secret will invalidate all existing user sessions${NC}"
        ;;

    2)
        # ============================================================
        # SERVICE KEY GENERATION
        # ============================================================

        echo ""
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${BLUE}           Service Key Generation${NC}"
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"

        # Prompt for environment
        echo ""
        echo -e "${BLUE}Select environment:${NC}"
        echo "1) Production (sk_live_)"
        echo "2) Development/Staging (sk_test_)"
        read -p "Enter choice [1-2]: " ENV_CHOICE

        case $ENV_CHOICE in
            1)
                ENV="live"
                ENV_NAME="Production"
                ;;
            2)
                ENV="test"
                ENV_NAME="Development/Staging"
                ;;
            *)
                echo -e "${RED}Invalid choice${NC}"
                exit 1
                ;;
        esac

        # Prompt for key name
        read -p "Enter a name for this service key (e.g., 'Backend Service', 'Cron Jobs'): " KEY_NAME

        # Prompt for description
        read -p "Enter a description (optional): " KEY_DESCRIPTION

        # Prompt for expiration
        echo -e "\n${BLUE}Set expiration (recommended for security):${NC}"
        echo "1) 90 days"
        echo "2) 1 year"
        echo "3) Never (not recommended)"
        read -p "Enter choice [1-3]: " EXPIRY_CHOICE

        case $EXPIRY_CHOICE in
            1)
                EXPIRY_SQL="NOW() + INTERVAL '90 days'"
                EXPIRY_DESC="90 days"
                ;;
            2)
                EXPIRY_SQL="NOW() + INTERVAL '1 year'"
                EXPIRY_DESC="1 year"
                ;;
            3)
                EXPIRY_SQL="NULL"
                EXPIRY_DESC="Never (⚠️ not recommended)"
                ;;
            *)
                echo -e "${RED}Invalid choice${NC}"
                exit 1
                ;;
        esac

        echo ""
        echo -e "${YELLOW}⚠️  IMPORTANT SECURITY WARNING ⚠️${NC}"
        echo -e "${YELLOW}Service keys bypass Row-Level Security and have FULL database access.${NC}"
        echo -e "${YELLOW}Never expose service keys to clients or commit them to version control.${NC}"
        echo ""
        read -p "Do you understand and wish to continue? (yes/no): " CONFIRM

        if [ "$CONFIRM" != "yes" ]; then
            echo -e "${RED}Aborted.${NC}"
            exit 0
        fi

        # Generate random key
        echo ""
        echo -e "${BLUE}Generating service key...${NC}"
        RANDOM_PART=$(openssl rand -base64 32 | tr -d '/+=\n')
        SERVICE_KEY="sk_${ENV}_${RANDOM_PART}"
        KEY_PREFIX="${SERVICE_KEY:0:8}"

        echo -e "${GREEN}✓ Service key generated${NC}"
        echo ""

        # Display the key
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}                    YOUR SERVICE KEY${NC}"
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        echo ""
        echo -e "${YELLOW}⚠️  SAVE THIS KEY NOW - IT WILL ONLY BE SHOWN ONCE ⚠️${NC}"
        echo ""
        echo -e "${BLUE}Service Key:${NC}"
        echo -e "${GREEN}${SERVICE_KEY}${NC}"
        echo ""
        echo -e "${BLUE}Key Prefix:${NC} ${KEY_PREFIX}"
        echo -e "${BLUE}Environment:${NC} ${ENV_NAME}"
        echo -e "${BLUE}Name:${NC} ${KEY_NAME}"
        echo -e "${BLUE}Description:${NC} ${KEY_DESCRIPTION:-N/A}"
        echo -e "${BLUE}Expires:${NC} ${EXPIRY_DESC}"
        echo ""
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        echo ""

        # Database insertion instructions
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${BLUE}           NEXT STEP: Store in Database${NC}"
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo ""
        echo -e "${BLUE}Run this SQL command:${NC}"
        echo ""
        cat <<SQL
INSERT INTO auth.service_keys (name, description, key_hash, key_prefix, enabled, expires_at)
VALUES (
    '${KEY_NAME}',
    '${KEY_DESCRIPTION}',
    crypt('${SERVICE_KEY}', gen_salt('bf', 12)),
    '${KEY_PREFIX}',
    true,
    ${EXPIRY_SQL}
);
SQL

        echo ""
        echo -e "${YELLOW}Store this key securely (environment variable, Kubernetes secret, etc.)${NC}"
        ;;

    3)
        # ============================================================
        # ANON KEY GENERATION (JWT)
        # ============================================================

        echo ""
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${BLUE}           Anonymous Key (JWT) Generation${NC}"
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo ""

        echo -e "${YELLOW}Note: Anon keys are JWT tokens with 'anon' role for client-side use.${NC}"
        echo -e "${YELLOW}They respect Row-Level Security policies.${NC}"
        echo ""

        # Prompt for JWT secret
        echo -e "${BLUE}Enter your Fluxbase JWT secret:${NC}"
        echo -e "${YELLOW}(This is the FLUXBASE_JWT_SECRET from your configuration)${NC}"
        read -s -p "JWT Secret: " JWT_SECRET
        echo ""

        if [ -z "$JWT_SECRET" ]; then
            echo -e "${RED}Error: JWT secret is required${NC}"
            exit 1
        fi

        # Prompt for expiration
        echo ""
        echo -e "${BLUE}Set token expiration:${NC}"
        echo "1) 1 hour"
        echo "2) 24 hours"
        echo "3) 7 days"
        echo "4) 1 year (not recommended for production)"
        echo "5) Custom"
        read -p "Enter choice [1-5]: " EXPIRY_CHOICE

        case $EXPIRY_CHOICE in
            1)
                EXPIRY_SECONDS=3600
                EXPIRY_DESC="1 hour"
                ;;
            2)
                EXPIRY_SECONDS=86400
                EXPIRY_DESC="24 hours"
                ;;
            3)
                EXPIRY_SECONDS=604800
                EXPIRY_DESC="7 days"
                ;;
            4)
                EXPIRY_SECONDS=31536000
                EXPIRY_DESC="1 year"
                ;;
            5)
                read -p "Enter expiration in seconds: " EXPIRY_SECONDS
                EXPIRY_DESC="${EXPIRY_SECONDS} seconds"
                ;;
            *)
                echo -e "${RED}Invalid choice${NC}"
                exit 1
                ;;
        esac

        # Generate JWT
        echo ""
        echo -e "${BLUE}Generating anonymous JWT token...${NC}"

        # Generate random user ID for anonymous user
        USER_ID=$(cat /proc/sys/kernel/random/uuid 2>/dev/null || uuidgen 2>/dev/null || openssl rand -hex 16)
        JTI=$(cat /proc/sys/kernel/random/uuid 2>/dev/null || uuidgen 2>/dev/null || openssl rand -hex 16)

        # JWT timestamps
        IAT=$(date +%s)
        EXP=$((IAT + EXPIRY_SECONDS))
        NBF=$IAT

        # Create JWT header
        HEADER='{"alg":"HS256","typ":"JWT"}'
        HEADER_B64=$(echo -n "$HEADER" | openssl base64 -e | tr -d '=' | tr '/+' '_-' | tr -d '\n')

        # Create JWT payload
        PAYLOAD=$(cat <<EOF
{
  "user_id": "${USER_ID}",
  "email": "",
  "role": "anon",
  "session_id": "",
  "token_type": "access",
  "is_anonymous": true,
  "iss": "fluxbase",
  "sub": "${USER_ID}",
  "iat": ${IAT},
  "exp": ${EXP},
  "nbf": ${NBF},
  "jti": "${JTI}"
}
EOF
)

        PAYLOAD_B64=$(echo -n "$PAYLOAD" | openssl base64 -e | tr -d '=' | tr '/+' '_-' | tr -d '\n')

        # Create signature
        SIGNATURE=$(echo -n "${HEADER_B64}.${PAYLOAD_B64}" | openssl dgst -sha256 -hmac "$JWT_SECRET" -binary | openssl base64 -e | tr -d '=' | tr '/+' '_-' | tr -d '\n')

        # Combine to create JWT
        ANON_KEY="${HEADER_B64}.${PAYLOAD_B64}.${SIGNATURE}"

        echo -e "${GREEN}✓ Anonymous key generated${NC}"
        echo ""

        # Display the key
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}                  YOUR ANONYMOUS KEY (JWT)${NC}"
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        echo ""
        echo -e "${BLUE}Anonymous Key:${NC}"
        echo -e "${GREEN}${ANON_KEY}${NC}"
        echo ""
        echo -e "${BLUE}User ID:${NC} ${USER_ID}"
        echo -e "${BLUE}Role:${NC} anon"
        echo -e "${BLUE}Expires:${NC} ${EXPIRY_DESC}"
        echo -e "${BLUE}Issued At:${NC} $(date -d @${IAT} 2>/dev/null || date -r ${IAT} 2>/dev/null)"
        echo -e "${BLUE}Expires At:${NC} $(date -d @${EXP} 2>/dev/null || date -r ${EXP} 2>/dev/null)"
        echo ""
        echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
        echo ""

        # Usage instructions
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${BLUE}           Usage Instructions${NC}"
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo ""

        echo -e "${BLUE}1. Environment Variable (Client-side):${NC}"
        cat <<'EOF'
# .env.local
VITE_FLUXBASE_URL=http://localhost:8080
VITE_FLUXBASE_ANON_KEY=<your-anon-key>
EOF
        echo ""

        echo -e "${BLUE}2. TypeScript SDK:${NC}"
        cat <<'EOF'
import { createClient } from "@fluxbase/client";

const fluxbase = createClient({
  url: import.meta.env.VITE_FLUXBASE_URL,
  anonKey: import.meta.env.VITE_FLUXBASE_ANON_KEY,
});

// Use for anonymous requests (respects RLS)
const { data } = await fluxbase.from("public_posts").select("*");
EOF
        echo ""

        echo -e "${BLUE}3. HTTP Requests:${NC}"
        cat <<'EOF'
curl -H "Authorization: Bearer <your-anon-key>" \
  http://localhost:8080/api/v1/tables/public_posts
EOF
        echo ""

        echo -e "${YELLOW}Note: Anon keys are safe for client-side use but still respect RLS policies.${NC}"
        echo -e "${YELLOW}Users will only see data allowed by your anon role policies.${NC}"
        echo ""
        ;;

    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac

# Final reminders
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}Key generation complete!${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo "For more information, see: docs/docs/guides/authentication.md"
echo ""
