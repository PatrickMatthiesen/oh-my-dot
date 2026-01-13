# Kubernetes kubectl Command Completion
# Enables kubectl shell completions for faster command-line usage

# Check if kubectl is installed
if command -v kubectl >/dev/null 2>&1; then
    # Generate completions for bash/zsh
    if [ -n "$BASH_VERSION" ]; then
        # Bash completion
        source <(kubectl completion bash)
        # Enable alias completion
        complete -o default -F __start_kubectl k
    elif [ -n "$ZSH_VERSION" ]; then
        # Zsh completion
        source <(kubectl completion zsh)
        # Enable alias completion
        compdef __start_kubectl k
    fi
    
    # Common kubectl aliases
    alias k='kubectl'
    alias kgp='kubectl get pods'
    alias kgs='kubectl get services'
    alias kgd='kubectl get deployments'
    alias kgn='kubectl get nodes'
    alias kdp='kubectl describe pod'
    alias kds='kubectl describe service'
    alias kdd='kubectl describe deployment'
    alias kaf='kubectl apply -f'
    alias kdelf='kubectl delete -f'
    alias kl='kubectl logs'
    alias klf='kubectl logs -f'
    alias kex='kubectl exec -it'
fi
