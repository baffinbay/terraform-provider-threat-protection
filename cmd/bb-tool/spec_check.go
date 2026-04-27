package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type SpecPaths struct {
	Paths map[string]map[string]interface{} `yaml:"paths"`
}

func handleSpecCheck(remoteURL, localPath string) {
	_ = context.Background()

	fetchData := func(uStr string) ([]byte, error) {
		req, err := http.NewRequest(http.MethodGet, uStr, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch %s: %s", uStr, resp.Status)
		}

		return io.ReadAll(resp.Body)
	}

	fmt.Printf("Fetching remote spec from %s...\n", remoteURL)
	remoteData, err := fetchData(remoteURL)
	if err != nil {
		fmt.Printf("Error fetching remote spec: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loading local spec from %s...\n", localPath)
	localData, err := os.ReadFile(localPath)
	if err != nil {
		fmt.Printf("Error reading local spec: %v\n", err)
		os.Exit(1)
	}

	var remoteSpec SpecPaths
	if err := yaml.Unmarshal(remoteData, &remoteSpec); err != nil {
		fmt.Printf("Error unmarshalling remote spec: %v\n", err)
		os.Exit(1)
	}

	var localSpec SpecPaths
	if err := yaml.Unmarshal(localData, &localSpec); err != nil {
		fmt.Printf("Error unmarshalling local spec: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n--- API Specification Comparison ---")

	diffFound := false

	remotePaths := getSortedKeys(remoteSpec.Paths)
	localPaths := getSortedKeys(localSpec.Paths)

	// Check for paths in remote but not in local
	for _, path := range remotePaths {
		remoteMethods := remoteSpec.Paths[path]
		localMethods := localSpec.Paths[path]

		if localMethods == nil {
			fmt.Printf("[NEW PATH] %s\n", path)
			diffFound = true
		} else {
			for method := range remoteMethods {
				if isMethod(method) {
					if _, ok := localMethods[method]; !ok {
						fmt.Printf("[NEW METHOD] %s %s\n", strings.ToUpper(method), path)
						diffFound = true
					}
				}
			}
		}
	}

	// Check for paths in local but not in remote
	for _, path := range localPaths {
		localMethods := localSpec.Paths[path]
		remoteMethods := remoteSpec.Paths[path]

		if remoteMethods == nil {
			fmt.Printf("[REMOVED PATH] %s\n", path)
			diffFound = true
		} else {
			for method := range localMethods {
				if isMethod(method) {
					if _, ok := remoteMethods[method]; !ok {
						fmt.Printf("[REMOVED METHOD] %s %s\n", strings.ToUpper(method), path)
						diffFound = true
					}
				}
			}
		}
	}

	if !diffFound {
		fmt.Println("No major differences found in paths and methods.")
	}

	fmt.Println("\n--- Client Implementation Validation ---")
	validateClientImplementation(remoteSpec.Paths)
}

func validateClientImplementation(paths map[string]map[string]interface{}) {
	clientDir := "internal/client"
	files, err := filepath.Glob(filepath.Join(clientDir, "*.go"))
	if err != nil {
		fmt.Printf("Error finding client files: %v\n", err)
		return
	}

	// Regex to find strings starting with /api/v2/
	pathRegex := regexp.MustCompile(`"/api/v2/[^"]+"`)

	allPathsInSpec := make(map[string]bool)
	for p := range paths {
		// Convert template paths like /api/v2/resource/{id} to regex-friendly pattern
		cleanPath := regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(p, "*")
		allPathsInSpec[cleanPath] = true
	}

	implementationError := false
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		matches := pathRegex.FindAllString(string(content), -1)
		for _, match := range matches {
			rawPath := strings.Trim(match, `"`)

			// If path ends with /, it's likely a concatenation like "/path/"+id
			lookupPath := rawPath
			if strings.HasSuffix(rawPath, "/") {
				lookupPath = rawPath + "*"
			}

			found := false
			for specPath := range allPathsInSpec {
				pattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(specPath), "\\*", "[^/]+") + "$"
				if ok, _ := regexp.MatchString(pattern, lookupPath); ok {
					found = true
					break
				}
				// Also try without the trailing slash if it's a fixed path
				if ok, _ := regexp.MatchString(pattern, strings.TrimSuffix(lookupPath, "*")); ok {
					found = true
					break
				}
			}

			if !found {
				fmt.Printf("[CLIENT ERROR] Path %s found in %s but not in remote spec\n", rawPath, file)
				implementationError = true
			}
		}
	}

	if !implementationError {
		fmt.Println("All paths found in client implementation are present in the remote spec.")
	}
}

func isMethod(m string) bool {
	switch strings.ToLower(m) {
	case "get", "post", "put", "patch", "delete", "options", "head":
		return true
	}
	return false
}

func getSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
