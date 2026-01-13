#!/usr/bin/env fish
# AWS CLI Command Completion for Fish
# Enables AWS CLI shell completions for faster command-line usage

# Check if aws is installed
if command -v aws >/dev/null 2>&1
    # AWS CLI completion for Fish
    # Fish completion is usually provided by the aws-cli package
    # or can be installed separately
    
    # If aws_completer is available, set up completion
    if command -v aws_completer >/dev/null 2>&1
        complete -c aws -f -a '(begin; set -lx COMP_SHELL fish; set -lx COMP_LINE (commandline); aws_completer; end)'
    end
    
    # Common AWS CLI aliases
    alias awswhoami='aws sts get-caller-identity'
    alias awsregions='aws ec2 describe-regions --output table'
    alias awsprofiles='aws configure list-profiles'
    
    # S3 aliases
    alias s3ls='aws s3 ls'
    alias s3cp='aws s3 cp'
    alias s3sync='aws s3 sync'
    alias s3rb='aws s3 rb'
    alias s3mb='aws s3 mb'
    
    # EC2 aliases
    alias ec2ls='aws ec2 describe-instances --query "Reservations[].Instances[].[InstanceId,State.Name,InstanceType,Tags[?Key==\'Name\'].Value|[0]]" --output table'
    alias ec2start='aws ec2 start-instances --instance-ids'
    alias ec2stop='aws ec2 stop-instances --instance-ids'
    
    # Lambda aliases
    alias lambdals='aws lambda list-functions --query "Functions[].[FunctionName,Runtime,LastModified]" --output table'
    
    # CloudFormation aliases
    alias cfnls='aws cloudformation list-stacks --stack-status-filter CREATE_COMPLETE UPDATE_COMPLETE --query "StackSummaries[].[StackName,StackStatus]" --output table'
end
