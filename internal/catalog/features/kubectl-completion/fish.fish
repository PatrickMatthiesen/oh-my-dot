#!/usr/bin/env fish
# Kubernetes kubectl Command Completion for Fish
# Enables kubectl shell completions for faster command-line usage

# Check if kubectl is installed
if command -v kubectl >/dev/null 2>&1
    # Generate Fish completions
    kubectl completion fish | source
    
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
end
