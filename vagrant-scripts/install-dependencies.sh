# docker
if ! command -v docker >/dev/null 2>&1; then
    sudo apt-get update
    sudo apt-get install ca-certificates curl
    sudo install -m 0755 -d /etc/apt/keyrings
    sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
    sudo chmod a+r /etc/apt/keyrings/docker.asc

    echo \
        "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
        $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" | \
        sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    
    sudo apt-get update
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    sudo usermod -aG docker $USER
else
    echo "docker found, skipping installation"
fi

# kind
if ! command -v kind >/dev/null 2>&1; then
    [ $(uname -m) = x86_64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.30.0/kind-linux-amd64
    chmod +x ./kind
    sudo mv ./kind /usr/local/bin/kind
else
    echo "kind found, skipping installation"
fi

# keadm
if ! command -v keadm >/dev/null 2>&1; then
    curl -Lo keadm-v1.21.0-linux-amd64.tar.gz https://github.com/kubeedge/kubeedge/releases/download/v1.21.0/keadm-v1.21.0-linux-amd64.tar.gz
    tar -xvzf keadm-v1.21.0-linux-amd64.tar.gz
    BIN_DIR=keadm-v1.21.0-linux-amd64
    sudo mv "./$BIN_DIR/keadm/keadm" /usr/local/bin/
    rm keadm-v1.21.0-linux-amd64.tar.gz
    rm -rf "./$BIN_DIR"
else
    echo "keadm found, skipping installation"
fi