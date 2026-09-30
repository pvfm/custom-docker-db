// Package envconfig discovers .env files and extracts the database settings
// used to configure the Postgres container.
package envconfig

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Candidates is the discovery order used when --env-file is not given.
var Candidates = []string{".env", ".env.local", ".env.development"}

const (
	DefaultUser     = "postgres"
	DefaultPassword = "postgres"
	DefaultPort     = 5432
)

// Config holds the settings for the container.
type Config struct {
	User     string
	Password string
	Name     string
	Port     int
	// Source is the env file the settings came from.
	Source string
}

// Load resolves the database config. With envFile set, only that file is
// read; otherwise Candidates are searched in dir. defaultName is used for the
// database name when the file does not set one (usually the directory name).
func Load(dir, envFile, defaultName string) (Config, error) {
	path, err := pickFile(dir, envFile)
	if err != nil {
		return Config{}, err
	}
	vars, err := godotenv.Read(path)
	if err != nil {
		return Config{}, fmt.Errorf("lendo %s: %w", path, err)
	}
	fields, err := dbFields(path, vars)
	if err != nil {
		return Config{}, err
	}
	cfg, err := build(fields, defaultName)
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	cfg.Source = path
	return cfg, nil
}

func pickFile(dir, envFile string) (string, error) {
	if envFile != "" {
		vars, err := godotenv.Read(envFile)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return "", fmt.Errorf("arquivo não encontrado: %s", envFile)
			}
			return "", fmt.Errorf("lendo %s: %w", envFile, err)
		}
		if !hasDBKeys(vars) {
			return "", &NoConfigError{File: envFile}
		}
		return envFile, nil
	}

	var found, searched []string
	for _, name := range Candidates {
		path := filepath.Join(dir, name)
		searched = append(searched, name)
		vars, err := godotenv.Read(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("lendo %s: %w", path, err)
		}
		if hasDBKeys(vars) {
			found = append(found, path)
		}
	}
	switch len(found) {
	case 0:
		return "", &NoConfigError{Searched: searched}
	case 1:
		return found[0], nil
	default:
		names := make([]string, len(found))
		for i, f := range found {
			names[i] = filepath.Base(f)
		}
		return "", &MultipleFilesError{Files: names}
	}
}

// split returns the field name (upper case, without prefix) of a DB_ or
// DATABASE_ key.
func split(key string) (field string, ok bool) {
	k := strings.ToUpper(key)
	for _, p := range []string{"DATABASE_", "DB_"} {
		if strings.HasPrefix(k, p) && len(k) > len(p) {
			return k[len(p):], true
		}
	}
	return "", false
}

func hasDBKeys(vars map[string]string) bool {
	for k := range vars {
		if _, ok := split(k); ok {
			return true
		}
	}
	return false
}

// dbFields returns field -> value, failing when a field is defined with both
// prefixes.
func dbFields(file string, vars map[string]string) (map[string]string, error) {
	fields := map[string]string{}
	seen := map[string][]string{}
	for k, v := range vars {
		f, ok := split(k)
		if !ok {
			continue
		}
		fields[f] = v
		seen[f] = append(seen[f], k)
	}
	var conflicts []string
	for _, keys := range seen {
		if len(keys) > 1 {
			sort.Strings(keys)
			conflicts = append(conflicts, keys...)
		}
	}
	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		return nil, &PrefixConflictError{File: file, Keys: conflicts}
	}
	return fields, nil
}

func first(fields map[string]string, names ...string) (string, bool) {
	for _, n := range names {
		if v, ok := fields[n]; ok {
			return v, true
		}
	}
	return "", false
}

func build(fields map[string]string, defaultName string) (Config, error) {
	cfg := Config{User: DefaultUser, Password: DefaultPassword, Name: defaultName, Port: DefaultPort}

	user, hasUser := first(fields, "USER", "USERNAME")
	pass, hasPass := first(fields, "PASSWORD", "PASS")
	name, hasName := first(fields, "NAME", "DATABASE")
	port, hasPort := first(fields, "PORT")

	if hasUser || hasPass || hasName || hasPort {
		if hasUser {
			cfg.User = user
		}
		if hasPass {
			cfg.Password = pass
		}
		if hasName {
			cfg.Name = name
		}
		if hasPort {
			p, err := parsePort(port)
			if err != nil {
				return Config{}, err
			}
			cfg.Port = p
		}
		return cfg, nil
	}

	if raw, ok := first(fields, "URL"); ok {
		return fromURL(raw, cfg)
	}
	// Only ignored fields (e.g. DB_HOST): defaults apply.
	return cfg, nil
}

func fromURL(raw string, cfg Config) (Config, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Config{}, fmt.Errorf("URL de banco inválida: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return Config{}, fmt.Errorf("banco não suportado: esquema %q (esperado postgres ou postgresql)", u.Scheme)
	}
	if u.User != nil {
		if n := u.User.Username(); n != "" {
			cfg.User = n
		}
		if p, ok := u.User.Password(); ok {
			cfg.Password = p
		}
	}
	if p := u.Port(); p != "" {
		port, err := parsePort(p)
		if err != nil {
			return Config{}, err
		}
		cfg.Port = port
	}
	if n := strings.TrimPrefix(u.Path, "/"); n != "" {
		cfg.Name = n
	}
	return cfg, nil
}

func parsePort(s string) (int, error) {
	p, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || p < 1 || p > 65535 {
		return 0, fmt.Errorf("porta inválida: %q", s)
	}
	return p, nil
}
