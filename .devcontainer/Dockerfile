# syntax=docker/dockerfile:1
FROM golang:1.25-bookworm

# Avoid warnings by switching to noninteractive
ENV DEBIAN_FRONTEND=noninteractive

# Install base dependencies
RUN apt-get update && apt-get install -y \
    git \
    curl \
    wget \
    vim \
    nano \
    htop \
    build-essential \
    pkg-config \
    postgresql-client \
    redis-tools \
    httpie \
    jq \
    tree \
    zip \
    unzip \
    sudo \
    locales \
    ca-certificates \
    gnupg \
    lsb-release \
    software-properties-common \
    zsh \
    lsof \
    && rm -rf /var/lib/apt/lists/*

# Set up locale
RUN echo "en_US.UTF-8 UTF-8" > /etc/locale.gen \
    && locale-gen en_US.UTF-8 \
    && update-locale LANG=en_US.UTF-8
ENV LANG=en_US.UTF-8 \
    LANGUAGE=en_US:en \
    LC_ALL=en_US.UTF-8

# Install Docker CLI
RUN install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get install -y docker-ce-cli docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js 20
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && npm install -g \
    typescript \
    @types/node \
    eslint \
    prettier \
    @swc/cli \
    @swc/core \
    tsx \
    nodemon \
    concurrently \
    npm-check-updates \
    serve \
    && rm -rf /var/lib/apt/lists/*

# Install k6 for load testing (using direct download to avoid keyserver issues)
RUN K6_VERSION=$(curl -s https://api.github.com/repos/grafana/k6/releases/latest | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/') \
    && curl -L "https://github.com/grafana/k6/releases/download/v${K6_VERSION}/k6-v${K6_VERSION}-linux-amd64.tar.gz" -o k6.tar.gz \
    && tar -xzf k6.tar.gz \
    && mv k6-v${K6_VERSION}-linux-amd64/k6 /usr/local/bin/ \
    && rm -rf k6.tar.gz k6-v${K6_VERSION}-linux-amd64 \
    && k6 version

# Create vscode user
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME -s /bin/zsh \
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

# Install Oh My Zsh for vscode user
USER $USERNAME
RUN sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended \
    && echo 'export PATH="/go/bin:$HOME/go/bin:$PATH"' >> ~/.zshrc \
    && echo 'export GOPATH="/go"' >> ~/.zshrc

# Switch back to root to install Go tools
USER root

# Install Go tools
RUN go install -v golang.org/x/tools/gopls@latest \
    && go install -v github.com/go-delve/delve/cmd/dlv@latest \
    && go install -v github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
    && go install -v github.com/air-verse/air@latest \
    && go install -v -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
    && go install -v github.com/swaggo/swag/cmd/swag@latest \
    && go install -v github.com/vektra/mockery/v2@latest \
    && go install -v github.com/onsi/ginkgo/v2/ginkgo@latest \
    && go install -v gotest.tools/gotestsum@latest \
    && go install -v github.com/rakyll/hey@latest \
    && go install -v github.com/securego/gosec/v2/cmd/gosec@latest \
    && go install -v github.com/mgechev/revive@latest \
    && go install -v github.com/kisielk/errcheck@latest \
    && go install -v honnef.co/go/tools/cmd/staticcheck@latest

# Set permissions for Go directories
RUN mkdir -p /go/pkg /go/bin \
    && chown -R $USERNAME:$USERNAME /go

# Set working directory
WORKDIR /workspace

# Set environment variables
ENV GOPATH=/go \
    PATH="/go/bin:${PATH}" \
    CGO_ENABLED=1 \
    GO111MODULE=on

# Switch to vscode user
USER $USERNAME

# Create Air config
RUN mkdir -p /home/$USERNAME/.air
COPY --chown=$USERNAME:$USERNAME <<EOF /home/$USERNAME/.air/.air.toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
args_bin = []
bin = "./tmp/main"
cmd = "go build -o ./tmp/main cmd/fluxbase/main.go"
delay = 1000
exclude_dir = ["assets", "tmp", "vendor", "testdata", "docs", "scripts", ".git", ".github", "dist", "node_modules"]
exclude_file = []
exclude_regex = ["_test.go"]
exclude_unchanged = false
follow_symlink = false
full_bin = ""
include_dir = []
include_ext = ["go", "tpl", "tmpl", "html", "yaml", "yml", "toml", "env"]
include_file = []
kill_delay = "0s"
log = "build-errors.log"
poll = false
poll_interval = 0
rerun = false
rerun_delay = 500
send_interrupt = false
stop_on_error = false

[color]
app = ""
build = "yellow"
main = "magenta"
runner = "green"
watcher = "cyan"

[log]
main_only = false
time = false

[misc]
clean_on_exit = false

[screen]
clear_on_rebuild = false
keep_scroll = true
EOF

# Switch back to noninteractive
ENV DEBIAN_FRONTEND=dialog
