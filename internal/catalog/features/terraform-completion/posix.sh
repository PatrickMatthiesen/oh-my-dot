# Terraform Command Completion
# Enables terraform shell completions for faster command-line usage

# Check if terraform is installed
if command -v terraform >/dev/null 2>&1; then
    # Terraform has built-in completion support
    if [ -n "$BASH_VERSION" ]; then
        # Bash completion
        complete -C "$(which terraform)" terraform
        complete -C "$(which terraform)" tf
    elif [ -n "$ZSH_VERSION" ]; then
        # Zsh completion
        autoload -U +X bashcompinit && bashcompinit
        complete -C "$(which terraform)" terraform
        complete -C "$(which terraform)" tf
    fi
    
    # Common terraform aliases
    alias tf='terraform'
    alias tfi='terraform init'
    alias tfp='terraform plan'
    alias tfa='terraform apply'
    alias tfd='terraform destroy'
    alias tfv='terraform validate'
    alias tff='terraform fmt'
    alias tfs='terraform show'
    alias tfw='terraform workspace'
    alias tfwsh='terraform workspace show'
    alias tfwl='terraform workspace list'
    alias tfws='terraform workspace select'
    alias tfo='terraform output'
    alias tfr='terraform refresh'
    alias tft='terraform taint'
    alias tfu='terraform untaint'
fi
