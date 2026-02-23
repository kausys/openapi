package scanner

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// processTypeAliases processes type alias declarations to map aliases to their original types.
// This is used to resolve enums when they are accessed through aliases (e.g., model.FeeType -> workspace.FeeType).
func (s *Scanner) processTypeAliases(filePath string, file *ast.File, pkg *packages.Package) {
	if pkg == nil || pkg.TypesInfo == nil {
		return
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Check if this is a type alias (uses = instead of type definition)
			if !typeSpec.Assign.IsValid() {
				continue
			}

			// Get the alias name with package prefix
			aliasName := typeSpec.Name.Name
			pkgName := pkg.Types.Name()
			fullAliasName := pkgName + "." + aliasName

			// Get the original type using types.Info
			obj := pkg.TypesInfo.Defs[typeSpec.Name]
			if obj == nil {
				continue
			}

			// Get the underlying named type
			originalType := s.resolveOriginalType(obj.Type())
			if originalType == "" {
				continue
			}

			// Store both the full alias name and the short alias name
			s.TypeAliases[fullAliasName] = originalType
			s.TypeAliases[aliasName] = originalType
		}
	}
}

// resolveOriginalType extracts the original type name from a types.Type.
func (s *Scanner) resolveOriginalType(t types.Type) string {
	// Handle named types
	if named, ok := t.(*types.Named); ok {
		obj := named.Obj()
		if obj == nil {
			return ""
		}

		pkg := obj.Pkg()
		if pkg == nil {
			return obj.Name()
		}

		return pkg.Name() + "." + obj.Name()
	}

	return ""
}

// ResolveTypeAlias resolves a type name to its original type if it's an alias.
// Returns the original type name, or the input if it's not an alias.
func (s *Scanner) ResolveTypeAlias(typeName string) string {
	return s.resolveTypeAliasWithVisited(typeName, make(map[string]bool))
}

// resolveTypeAliasWithVisited resolves type aliases with cycle detection.
func (s *Scanner) resolveTypeAliasWithVisited(typeName string, visited map[string]bool) string {
	if visited[typeName] {
		return typeName // cycle detected
	}
	visited[typeName] = true

	if original, ok := s.TypeAliases[typeName]; ok {
		return s.resolveTypeAliasWithVisited(original, visited)
	}

	if idx := strings.LastIndex(typeName, "."); idx >= 0 {
		shortName := typeName[idx+1:]
		if original, ok := s.TypeAliases[shortName]; ok {
			return s.resolveTypeAliasWithVisited(original, visited)
		}
	}

	return typeName
}

// GetEnumForType finds the enum info for a type, resolving aliases if necessary.
func (s *Scanner) GetEnumForType(typeName string) *EnumInfo {
	// Try direct lookup first
	if enumName, ok := s.TypeToEnum[typeName]; ok {
		if enumInfo, exists := s.Enums[enumName]; exists {
			return enumInfo
		}
	}

	// Try with short name
	shortName := typeName
	if idx := strings.LastIndex(typeName, "."); idx >= 0 {
		shortName = typeName[idx+1:]
	}

	if enumName, ok := s.TypeToEnum[shortName]; ok {
		if enumInfo, exists := s.Enums[enumName]; exists {
			return enumInfo
		}
	}

	// Try resolving as an alias
	resolved := s.ResolveTypeAlias(typeName)
	if resolved != typeName {
		return s.GetEnumForType(resolved)
	}

	return nil
}
