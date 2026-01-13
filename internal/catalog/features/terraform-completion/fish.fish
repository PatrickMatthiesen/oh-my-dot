# Terraform Command Completion for Fish
# Enables terraform shell completions for faster command-line usage

# Check if terraform is installed
if command -v terraform >/dev/null 2>&1
    # Terraform has built-in completion support
    # Fish completion is usually installed automatically with terraform
    # But we can ensure it's available
    complete -c terraform -a '(terraform -help 2>&1 | grep -E "^\s+[a-z]+" | awk "{print \$1}")'
    
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
    alias tfws='terraform workspace show'
    alias tfwl='terraform workspace list'
    alias tfwselect='terraform workspace select'
    alias tfo='terraform output'
    alias tfr='terraform refresh'
    alias tft='terraform taint'
    alias tfu='terraform untaint'
end
