package envconfig

import (
	"fmt"
	"strings"
)

// NoConfigError means no file with DB_/DATABASE_ variables was found.
// The caller (M3) offers to create the default config.
type NoConfigError struct {
	// File is the explicit --env-file path, or empty for automatic discovery.
	File string
	// Searched lists the files looked at during automatic discovery.
	Searched []string
}

func (e *NoConfigError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("%s não tem variáveis de banco (DB_* ou DATABASE_*)", e.File)
	}
	return fmt.Sprintf("nenhuma variável de banco (DB_* ou DATABASE_*) em %s", strings.Join(e.Searched, ", "))
}

// MultipleFilesError means more than one candidate file has database config.
type MultipleFilesError struct {
	Files []string
}

func (e *MultipleFilesError) Error() string {
	return fmt.Sprintf("config de banco encontrada em mais de um arquivo: %s; renomeie os que não devem ser usados ou use --env-file",
		strings.Join(e.Files, ", "))
}

// PrefixConflictError means the same field is set as both DB_X and DATABASE_X.
type PrefixConflictError struct {
	File string
	Keys []string
}

func (e *PrefixConflictError) Error() string {
	return fmt.Sprintf("%s define o mesmo campo com DB_ e DATABASE_ (%s); deixe apenas um",
		e.File, strings.Join(e.Keys, ", "))
}
