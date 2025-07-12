# FLEXT Project Makefile
.PHONY: help install test lint format build clean

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

install: ## Install dependencies
	poetry install --all-extras

test: ## Run tests
	poetry run pytest -v

lint: ## Run linting
	poetry run ruff check src tests
	poetry run mypy src

format: ## Format code
	poetry run ruff format src tests
	poetry run black src tests

check: lint type-check test ## Run all quality checks (lint, type-check, test)
	@echo "✅ All quality checks passed!"

type-check: ## Run type checking with mypy
	@echo "🔍 Running type checking for flexcore..."
	poetry run mypy src

build: ## Build package
	poetry build

clean: ## Clean artifacts
	rm -rf build/ dist/ *.egg-info/ .pytest_cache/ .mypy_cache/ .ruff_cache/
