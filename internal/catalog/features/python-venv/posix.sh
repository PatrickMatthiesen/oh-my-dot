#!/usr/bin/env sh
# Python Virtual Environment Helpers
# Useful functions and aliases for working with Python virtual environments

# Function to create a new virtual environment
venv() {
    if [ $# -eq 0 ]; then
        # No arguments, create venv in current directory
        python3 -m venv venv
        echo "Created virtual environment 'venv'"
        echo "Activate with: source venv/bin/activate"
    else
        # Create with specified name
        python3 -m venv "$1"
        echo "Created virtual environment '$1'"
        echo "Activate with: source $1/bin/activate"
    fi
}

# Function to activate common venv names
activate() {
    if [ -f "venv/bin/activate" ]; then
        . venv/bin/activate
    elif [ -f ".venv/bin/activate" ]; then
        . .venv/bin/activate
    elif [ -f "env/bin/activate" ]; then
        . env/bin/activate
    elif [ $# -eq 1 ] && [ -f "$1/bin/activate" ]; then
        . "$1/bin/activate"
    else
        echo "No virtual environment found. Tried: venv, .venv, env"
        return 1
    fi
}

# Alias to deactivate (if venv is active)
alias deact='deactivate 2>/dev/null || echo "No active virtual environment"'

# Alias for quick pip install
alias pipi='pip install'
alias pipir='pip install -r requirements.txt'
alias pipu='pip install --upgrade'
alias pipun='pip uninstall'
alias pipf='pip freeze'
alias pipfr='pip freeze > requirements.txt'
alias pipl='pip list'

# Python aliases
alias py='python3'
alias python='python3'
alias ipy='ipython'

# Django aliases (if using Django)
alias djrun='python manage.py runserver'
alias djmig='python manage.py migrate'
alias djmake='python manage.py makemigrations'
alias djshell='python manage.py shell'
alias djtest='python manage.py test'

# Flask aliases (if using Flask)
alias flaskrun='flask run'
alias flaskshell='flask shell'
