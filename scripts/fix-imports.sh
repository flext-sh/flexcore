#!/bin/bash

# Script para corrigir imports após reorganização

echo "Corrigindo imports..."

# Substitui imports antigos pelos novos
find . -name "*.go" -exec sed -i \
	-e 's|github.com/flext-sh/flexcore/shared/errors|github.com/flext-sh/flexcore/pkg/errors|g' \
	-e 's|github.com/flext-sh/flexcore/shared/result|github.com/flext-sh/flexcore/pkg/result|g' \
	-e 's|github.com/flext-sh/flexcore/shared/patterns|github.com/flext-sh/flexcore/pkg/patterns|g' \
	-e 's|github.com/flext-sh/flexcore/domain/entities|github.com/flext-sh/flexcore/internal/domain/entities|g' \
	-e 's|github.com/flext-sh/flexcore/domain|github.com/flext-sh/flexcore/internal/domain|g' \
	-e 's|github.com/flext-sh/flexcore/application/commands|github.com/flext-sh/flexcore/internal/app/commands|g' \
	-e 's|github.com/flext-sh/flexcore/application/queries|github.com/flext-sh/flexcore/internal/app/queries|g' \
	-e 's|github.com/flext-sh/flexcore/plugins/plugins|github.com/flext-sh/flexcore/internal/infrastructure/plugins|g' \
	{} \;

echo "Imports corrigidos!"
