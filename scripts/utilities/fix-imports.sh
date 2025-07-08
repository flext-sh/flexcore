#!/bin/bash

# Script para corrigir imports após reorganização

echo "Corrigindo imports..."

# Substitui imports antigos pelos novos
find . -name "*.go" -exec sed -i \
  -e 's|github.com/flext/flexcore/shared/errors|github.com/flext/flexcore/pkg/errors|g' \
  -e 's|github.com/flext/flexcore/shared/result|github.com/flext/flexcore/pkg/result|g' \
  -e 's|github.com/flext/flexcore/shared/patterns|github.com/flext/flexcore/pkg/patterns|g' \
  -e 's|github.com/flext/flexcore/domain/entities|github.com/flext/flexcore/internal/domain/entities|g' \
  -e 's|github.com/flext/flexcore/domain|github.com/flext/flexcore/internal/domain|g' \
  -e 's|github.com/flext/flexcore/application/commands|github.com/flext/flexcore/internal/app/commands|g' \
  -e 's|github.com/flext/flexcore/application/queries|github.com/flext/flexcore/internal/app/queries|g' \
  -e 's|github.com/flext/flexcore/plugins/plugins|github.com/flext/flexcore/internal/infrastructure/plugins|g' \
  {} \;

echo "Imports corrigidos!"