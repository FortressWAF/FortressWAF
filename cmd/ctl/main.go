package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var (
	apiURL  string
	apiKey  string
	timeout time.Duration
)

func main() {
	rootCmd := &cobra.Command{
		Use:          "fortressctl",
		Short:        "FortressWAF CLI management tool",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if envURL := os.Getenv("FORTRESS_API_URL"); envURL != "" && !cmd.Flags().Changed("api-url") {
				apiURL = envURL
			}
			if envKey := os.Getenv("FORTRESS_API_KEY"); envKey != "" && !cmd.Flags().Changed("api-key") {
				apiKey = envKey
			}
			if !strings.HasPrefix(apiURL, "http") {
				apiURL = "http://" + apiURL
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "localhost:8443", "WAF API URL (or FORTRESS_API_URL)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for auth (or FORTRESS_API_KEY)")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 10*time.Second, "HTTP request timeout")

	rootCmd.AddCommand(
		siteCmd(),
		ruleCmd(),
		logsCmd(),
		patchCmd(),
		configCmd(),
		statusCmd(),
		versionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRequest(method, path string, body io.Reader) (*http.Request, error) {
	u := fmt.Sprintf("%s/api/v1%s", apiURL, path)
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := newRequest(method, path, reader)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: timeout}
	return client.Do(req)
}

func doRequestRaw(method, path string, body io.Reader) (*http.Response, error) {
	req, err := newRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: timeout}
	return client.Do(req)
}

func handleResponse(resp *http.Response, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			fmt.Fprintf(os.Stderr, "Error %d: %s\n", resp.StatusCode, string(body))
			os.Exit(1)
		}
		if len(body) > 0 {
			fmt.Println(string(body))
		}
		return
	}

	if resp.StatusCode >= 400 {
		msg, _ := result["error"].(string)
		detail, _ := result["detail"].(string)
		fmt.Fprintf(os.Stderr, "Error %d: %s", resp.StatusCode, msg)
		if detail != "" {
			fmt.Fprintf(os.Stderr, " - %s", detail)
		}
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}

	pretty, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(pretty))
}

func siteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "site",
		Short: "Manage protected sites",
	}

	var name, domain, origin string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a protected site",
		Run: func(cmd *cobra.Command, args []string) {
			if name == "" {
				fmt.Fprintln(os.Stderr, "Error: --name is required")
				os.Exit(1)
			}
			if domain == "" {
				fmt.Fprintln(os.Stderr, "Error: --domain is required")
				os.Exit(1)
			}
			if origin == "" {
				fmt.Fprintln(os.Stderr, "Error: --origin is required")
				os.Exit(1)
			}

			body := map[string]interface{}{
				"name":        name,
				"domains":     strings.Split(domain, ","),
				"upstream":    origin,
				"waf_enabled": true,
			}
			resp, err := doRequest("POST", "/sites", body)
			handleResponse(resp, err)
		},
	}
	addCmd.Flags().StringVar(&name, "name", "", "site name")
	addCmd.Flags().StringVar(&domain, "domain", "", "domain(s), comma-separated")
	addCmd.Flags().StringVar(&origin, "origin", "", "origin URL")
	cmd.AddCommand(addCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all sites",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := doRequest("GET", "/sites", nil)
			handleResponse(resp, err)
		},
	}
	cmd.AddCommand(listCmd)

	var removeName string
	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a site",
		Run: func(cmd *cobra.Command, args []string) {
			if removeName == "" {
				fmt.Fprintln(os.Stderr, "Error: --name is required")
				os.Exit(1)
			}
			resp, err := doRequest("DELETE", "/sites/"+removeName, nil)
			handleResponse(resp, err)
		},
	}
	removeCmd.Flags().StringVar(&removeName, "name", "", "site name to remove")
	cmd.AddCommand(removeCmd)

	return cmd
}

func ruleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rule",
		Short: "Manage WAF rules",
	}

	var ruleFile string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a rule from YAML file",
		Run: func(cmd *cobra.Command, args []string) {
			if ruleFile == "" {
				fmt.Fprintln(os.Stderr, "Error: --file is required")
				os.Exit(1)
			}

			data, err := os.ReadFile(ruleFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
				os.Exit(1)
			}

			contentType := "application/json"
			if strings.HasSuffix(strings.ToLower(ruleFile), ".yaml") || strings.HasSuffix(strings.ToLower(ruleFile), ".yml") {
				contentType = "application/x-yaml"
			}

			req, err := newRequest("POST", "/rules", bytes.NewReader(data))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			req.Header.Set("Content-Type", contentType)

			client := &http.Client{Timeout: timeout}
			resp, err := client.Do(req)
			handleResponse(resp, err)
		},
	}
	createCmd.Flags().StringVar(&ruleFile, "file", "", "path to rule YAML file")
	cmd.AddCommand(createCmd)

	var severity, tag string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List rules",
		Run: func(cmd *cobra.Command, args []string) {
			path := "/rules"
			qs := []string{}
			if severity != "" {
				qs = append(qs, "severity="+severity)
			}
			if tag != "" {
				qs = append(qs, "tag="+tag)
			}
			if len(qs) > 0 {
				path += "?" + strings.Join(qs, "&")
			}
			resp, err := doRequest("GET", path, nil)
			handleResponse(resp, err)
		},
	}
	listCmd.Flags().StringVar(&severity, "severity", "", "filter by severity")
	listCmd.Flags().StringVar(&tag, "tag", "", "filter by tag")
	cmd.AddCommand(listCmd)

	var ruleID string
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a rule",
		Run: func(cmd *cobra.Command, args []string) {
			if ruleID == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				os.Exit(1)
			}
			resp, err := doRequest("DELETE", "/rules/"+ruleID, nil)
			handleResponse(resp, err)
		},
	}
	deleteCmd.Flags().StringVar(&ruleID, "id", "", "rule ID to delete")
	cmd.AddCommand(deleteCmd)

	var requestFile string
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test a rule against a sample request",
		Run: func(cmd *cobra.Command, args []string) {
			if ruleID == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				os.Exit(1)
			}
			if requestFile == "" {
				fmt.Fprintln(os.Stderr, "Error: --request is required")
				os.Exit(1)
			}

			data, err := os.ReadFile(requestFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading request file: %v\n", err)
				os.Exit(1)
			}

			var reqBody interface{}
			if err := json.Unmarshal(data, &reqBody); err != nil {
				reqBody = map[string]interface{}{
					"request": map[string]interface{}{
						"raw": string(data),
					},
				}
			}

			resp, err := doRequest("POST", "/rules/"+ruleID+"/test", reqBody)
			handleResponse(resp, err)
		},
	}
	testCmd.Flags().StringVar(&ruleID, "id", "", "rule ID to test")
	testCmd.Flags().StringVar(&requestFile, "request", "", "path to request YAML file")
	cmd.AddCommand(testCmd)

	return cmd
}

func logsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View and export logs",
	}

	var site string
	var limit int
	tailCmd := &cobra.Command{
		Use:   "tail",
		Short: "Tail recent logs",
		Run: func(cmd *cobra.Command, args []string) {
			path := "/logs/tail"
			qs := []string{}
			if site != "" {
				qs = append(qs, "site="+site)
			}
			if limit > 0 {
				qs = append(qs, fmt.Sprintf("limit=%d", limit))
			}
			if len(qs) > 0 {
				path += "?" + strings.Join(qs, "&")
			}
			resp, err := doRequest("GET", path, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				body, _ := io.ReadAll(resp.Body)
				fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
				os.Exit(1)
			}

			entries, ok := result["logs"].([]interface{})
			if !ok {
				entries, ok = result["entries"].([]interface{})
			}
			if !ok {
				pretty, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(pretty))
				return
			}

			for _, entry := range entries {
				pretty, _ := json.MarshalIndent(entry, "", "  ")
				fmt.Println(string(pretty))
				fmt.Println("---")
			}
		},
	}
	tailCmd.Flags().StringVar(&site, "site", "", "filter by site")
	tailCmd.Flags().IntVar(&limit, "limit", 50, "number of log entries")
	cmd.AddCommand(tailCmd)

	var format, output string
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export logs to file",
		Run: func(cmd *cobra.Command, args []string) {
			if format == "" {
				format = "json"
			}
			if output == "" {
				output = fmt.Sprintf("fortress-logs-%s.%s", time.Now().Format("20060102-150405"), format)
			}

			path := "/logs/export?format=" + format
			resp, err := doRequestRaw("GET", path, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				body, _ := io.ReadAll(resp.Body)
				fmt.Fprintf(os.Stderr, "Error %d: %s\n", resp.StatusCode, string(body))
				os.Exit(1)
			}

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
				os.Exit(1)
			}

			if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil && !os.IsExist(err) {
				fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
				os.Exit(1)
			}

			if err := os.WriteFile(output, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Exported %d bytes to %s\n", len(data), output)
		},
	}
	exportCmd.Flags().StringVar(&format, "format", "json", "export format: json or csv")
	exportCmd.Flags().StringVar(&output, "output", "", "output file path")
	cmd.AddCommand(exportCmd)

	return cmd
}

func patchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch",
		Short: "Manage virtual patches",
	}

	var cve string
	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a virtual patch",
		Run: func(cmd *cobra.Command, args []string) {
			if cve == "" {
				fmt.Fprintln(os.Stderr, "Error: --cve is required")
				os.Exit(1)
			}
			body := map[string]interface{}{
				"cve": cve,
			}
			resp, err := doRequest("POST", "/patches/"+cve+"/apply", body)
			handleResponse(resp, err)
		},
	}
	applyCmd.Flags().StringVar(&cve, "cve", "", "CVE ID to patch")
	cmd.AddCommand(applyCmd)

	revokeCmd := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke a virtual patch",
		Run: func(cmd *cobra.Command, args []string) {
			if cve == "" {
				fmt.Fprintln(os.Stderr, "Error: --cve is required")
				os.Exit(1)
			}
			resp, err := doRequest("POST", "/patches/"+cve+"/revoke", nil)
			handleResponse(resp, err)
		},
	}
	revokeCmd.Flags().StringVar(&cve, "cve", "", "CVE ID to revoke")
	cmd.AddCommand(revokeCmd)

	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage WAF configuration",
	}

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate current configuration",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := doRequest("POST", "/config/validate", nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				body, _ := io.ReadAll(resp.Body)
				fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
				os.Exit(1)
			}

			valid, _ := result["valid"].(bool)
			if valid {
				fmt.Println("Configuration is valid")
			} else {
				fmt.Println("Configuration is INVALID")
				if errs, ok := result["errors"].([]interface{}); ok {
					for _, e := range errs {
						fmt.Printf("  - %v\n", e)
					}
				}
				if errs, ok := result["errors"].(interface{}); ok && !valid {
					pretty, _ := json.MarshalIndent(errs, "  ", "  ")
					fmt.Printf("  %s\n", string(pretty))
				}
				os.Exit(1)
			}
		},
	}
	cmd.AddCommand(validateCmd)

	var outputFile string
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export configuration to YAML file",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := doRequestRaw("GET", "/config/export", nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				body, _ := io.ReadAll(resp.Body)
				fmt.Fprintf(os.Stderr, "Error %d: %s\n", resp.StatusCode, string(body))
				os.Exit(1)
			}

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
				os.Exit(1)
			}

			if outputFile == "" {
				fmt.Println(string(data))
				return
			}

			if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil && !os.IsExist(err) {
				fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
				os.Exit(1)
			}

			if err := os.WriteFile(outputFile, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Configuration exported to %s (%d bytes)\n", outputFile, len(data))
		},
	}
	exportCmd.Flags().StringVar(&outputFile, "output", "", "output file path (default: stdout)")
	cmd.AddCommand(exportCmd)

	var diffFile string
	diffCmd := &cobra.Command{
		Use:   "diff",
		Short: "Show diff between current and proposed configuration",
		Run: func(cmd *cobra.Command, args []string) {
			if diffFile == "" {
				fmt.Fprintln(os.Stderr, "Error: --file is required")
				os.Exit(1)
			}

			data, err := os.ReadFile(diffFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
				os.Exit(1)
			}

			contentType := "application/json"
			if strings.HasSuffix(strings.ToLower(diffFile), ".yaml") || strings.HasSuffix(strings.ToLower(diffFile), ".yml") {
				contentType = "application/x-yaml"
			}

			req, err := newRequest("POST", "/config/diff", bytes.NewReader(data))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			req.Header.Set("Content-Type", contentType)

			client := &http.Client{Timeout: timeout}
			resp, err := client.Do(req)
			handleResponse(resp, err)
		},
	}
	diffCmd.Flags().StringVar(&diffFile, "file", "", "path to proposed config file")
	cmd.AddCommand(diffCmd)

	var applyFile string
	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply new configuration",
		Run: func(cmd *cobra.Command, args []string) {
			if applyFile == "" {
				fmt.Fprintln(os.Stderr, "Error: --file is required")
				os.Exit(1)
			}

			data, err := os.ReadFile(applyFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
				os.Exit(1)
			}

			contentType := "application/json"
			if strings.HasSuffix(strings.ToLower(applyFile), ".yaml") || strings.HasSuffix(strings.ToLower(applyFile), ".yml") {
				contentType = "application/x-yaml"
			}

			req, err := newRequest("POST", "/config/import", bytes.NewReader(data))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			req.Header.Set("Content-Type", contentType)

			client := &http.Client{Timeout: timeout}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				body, _ := io.ReadAll(resp.Body)
				fmt.Fprintf(os.Stderr, "Error %d: %s\n", resp.StatusCode, string(body))
				os.Exit(1)
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				fmt.Println("Configuration applied successfully")
				return
			}

			pretty, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(pretty))
		},
	}
	applyCmd.Flags().StringVar(&applyFile, "file", "", "path to new config file")
	cmd.AddCommand(applyCmd)

	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show WAF status",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := doRequest("GET", "/status", nil)
			handleResponse(resp, err)
		},
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show fortressctl version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("fortressctl %s\n", Version)
			fmt.Printf("  commit:     %s\n", Commit)
			fmt.Printf("  build date: %s\n", BuildDate)
			fmt.Printf("  go version: %s\n", strings.TrimPrefix(flag.Lookup("go").Value.String(), "go"))
		},
	}
}

var _ = io.EOF
var _ = json.Marshal
