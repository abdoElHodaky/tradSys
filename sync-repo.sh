#!/bin/bash
# Repository synchronization script

# Fetch and remove deleted remote references
git fetch --prune

# Checkout main branch
git checkout main

# Pull latest changes
git pull origin main

# Clean local branches except main
git branch | grep -v "main" | xargs git branch -D

echo "Repository synchronized successfully!"
echo "Main branch is up-to-date and all other local branches have been removed."

