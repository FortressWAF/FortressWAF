package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
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

const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiRed     = "\033[31m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiBlue    = "\033[34m"
	ansiMagenta = "\033[35m"
	ansiCyan    = "\033[36m"
	ansiWhite   = "\033[37m"
	ansiGray    = "\033[90m"
	ansiClear   = "\033[2J\033[H"
	ansiHide    = "\033[?25l"
	ansiShow    = "\033[?25h"
)

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, ansiRed+"Error: "+format+ansiReset+"\n", args...)
	os.Exit(1)
}

func main() {
	rootCmd := &cobra.Command{
		Use:          "fortressctl",
		Short:        "FortressWAF CLI management tool",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
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
		monitorCmd(),
		statsCmd(),
		topCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	u := fmt.Sprintf("%s/api/v1%s", apiURL, path)
	req, err := http.NewRequestWithContext(ctx, method, u, body)
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
	req, err := newRequest(context.Background(), method, path, reader)
	if err != nil {
		return nil, err
	}
	c := &http.Client{Timeout: timeout}
	return c.Do(req)
}

func doRequestRaw(method, path string, body io.Reader) (*http.Response, error) {
	req, err := newRequest(context.Background(), method, path, body)
	if err != nil {
		return nil, err
	}
	c := &http.Client{Timeout: timeout}
	return c.Do(req)
}

// ------- helpers -------

func bold(s string) string   { return ansiBold + s + ansiReset }
func red(s string) string    { return ansiRed + s + ansiReset }
func green(s string) string  { return ansiGreen + s + ansiReset }
func yellow(s string) string { return ansiYellow + s + ansiReset }
func blue(s string) string   { return ansiBlue + s + ansiReset }
func cyan(s string) string   { return ansiCyan + s + ansiReset }
func gray(s string) string   { return ansiGray + s + ansiReset }

func bar(pct float64, width int) string {
	filled := int(pct * float64(width) / 100)
	if filled > width {
		filled = width
	}
	b := strings.Builder{}
	for i := 0; i < width; i++ {
		if i < filled {
			b.WriteString("█")
		} else {
			b.WriteString("░")
		}
	}
	return b.String()
}

func colorForPct(pct float64) string {
	switch {
	case pct >= 90:
		return ansiRed
	case pct >= 60:
		return ansiYellow
	default:
		return ansiGreen
	}
}

func coloredBar(pct float64, width int) string {
	c := colorForPct(pct)
	return c + bar(pct, width) + ansiReset
}

func apiGetRaw(path string, params map[string]string) (map[string]interface{}, error) {
	qs := []string{}
	for k, v := range params {
		qs = append(qs, k+"="+v)
	}
	if len(qs) > 0 {
		path += "?" + strings.Join(qs, "&")
	}
	resp, err := doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if resp.StatusCode >= 400 {
		msg, _ := result["error"].(string)
		return nil, fmt.Errorf("%s (HTTP %d)", msg, resp.StatusCode)
	}
	return result, nil
}

func apiGet(path string, params map[string]string) (map[string]interface{}, error) {
	resp, err := apiGetRaw(path, params)
	if err != nil {
		return nil, err
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		data = resp
	}
	return data, nil
}

func handleResponse(resp *http.Response, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, ansiRed+"Error: %v"+ansiReset+"\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			fmt.Fprintf(os.Stderr, ansiRed+"Error %d: %s"+ansiReset+"\n", resp.StatusCode, string(body))
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
		fmt.Fprintf(os.Stderr, ansiRed+"Error %d: %s", resp.StatusCode, msg)
		if detail != "" {
			fmt.Fprintf(os.Stderr, " - %s", detail)
		}
		fmt.Fprintln(os.Stderr, ansiReset)
		os.Exit(1)
	}

	pretty, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(pretty))
}

func printTable(header []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Println(gray("  (no data)"))
		return
	}
	cols := len(header)
	widths := make([]int, cols)
	for i, h := range header {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, v := range row {
			if len(v) > widths[i] {
				widths[i] = len(v)
			}
		}
	}
	total := 1
	for _, w := range widths {
		total += w + 3
	}
	sep := strings.Repeat("─", total)

	fmt.Println(cyan(sep))
	line := "│"
	for i, h := range header {
		line += " " + bold(h) + strings.Repeat(" ", widths[i]-len(h)) + " │"
	}
	fmt.Println(line)
	fmt.Println(cyan(sep))
	for _, row := range rows {
		line := "│"
		for i, v := range row {
			line += " " + v + strings.Repeat(" ", widths[i]-len(v)) + " │"
		}
		fmt.Println(line)
	}
	fmt.Println(cyan(sep))
}

// ------- commands -------

func siteCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "site", Short: "Manage protected sites"}
	var name, domain, origin string

	addCmd := &cobra.Command{
		Use: "add", Short: "Add a protected site",
		Run: func(cmd *cobra.Command, args []string) {
			if name == "" {
				fatal("--name is required")
			}
			if domain == "" {
				fatal("--domain is required")
			}
			if origin == "" {
				fatal("--origin is required")
			}
			body := map[string]interface{}{
				"name": name, "domains": strings.Split(domain, ","),
				"upstream": origin, "waf_enabled": true,
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
		Use: "list", Short: "List all sites",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := doRequest("GET", "/sites", nil)
			handleResponse(resp, err)
		},
	}
	cmd.AddCommand(listCmd)

	var removeName string
	removeCmd := &cobra.Command{
		Use: "remove", Short: "Remove a site",
		Run: func(cmd *cobra.Command, args []string) {
			if removeName == "" {
				fatal("--name is required")
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
	cmd := &cobra.Command{Use: "rule", Short: "Manage WAF rules"}
	var ruleFile, severity, tag, ruleID, requestFile string

	createCmd := &cobra.Command{
		Use: "create", Short: "Create a rule from YAML file",
		Run: func(cmd *cobra.Command, args []string) {
			if ruleFile == "" {
				fatal("--file is required")
			}
			data, err := os.ReadFile(ruleFile)
			if err != nil {
				fatal("reading file: %v", err)
			}
			contentType := "application/json"
			if strings.HasSuffix(strings.ToLower(ruleFile), ".yaml") || strings.HasSuffix(strings.ToLower(ruleFile), ".yml") {
				contentType = "application/x-yaml"
			}
			req, err := newRequest(context.Background(), "POST", "/rules", bytes.NewReader(data))
			if err != nil {
				fatal("%v", err)
			}
			req.Header.Set("Content-Type", contentType)
			client := &http.Client{Timeout: timeout}
			resp, err := client.Do(req)
			handleResponse(resp, err)
		},
	}
	createCmd.Flags().StringVar(&ruleFile, "file", "", "path to rule YAML file")
	cmd.AddCommand(createCmd)

	listCmd := &cobra.Command{
		Use: "list", Short: "List rules",
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

	deleteCmd := &cobra.Command{
		Use: "delete", Short: "Delete a rule",
		Run: func(cmd *cobra.Command, args []string) {
			if ruleID == "" {
				fatal("--id is required")
			}
			resp, err := doRequest("DELETE", "/rules/"+ruleID, nil)
			handleResponse(resp, err)
		},
	}
	deleteCmd.Flags().StringVar(&ruleID, "id", "", "rule ID to delete")
	cmd.AddCommand(deleteCmd)

	testCmd := &cobra.Command{
		Use: "test", Short: "Test a rule against a sample request",
		Run: func(cmd *cobra.Command, args []string) {
			if ruleID == "" {
				fatal("--id is required")
			}
			if requestFile == "" {
				fatal("--request is required")
			}
			data, err := os.ReadFile(requestFile)
			if err != nil {
				fatal("reading request file: %v", err)
			}
			var reqBody interface{}
			if err := json.Unmarshal(data, &reqBody); err != nil {
				reqBody = map[string]interface{}{"request": map[string]interface{}{"raw": string(data)}}
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
	cmd := &cobra.Command{Use: "logs", Short: "View and export logs"}
	var site string
	var limit int
	tailCmd := &cobra.Command{
		Use: "tail", Short: "Tail recent logs",
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
				fatal("%v", err)
			}
			defer resp.Body.Close()
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				body, _ := io.ReadAll(resp.Body)
				fatal("%s", string(body))
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
		Use: "export", Short: "Export logs to file",
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
				fatal("%v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				body, _ := io.ReadAll(resp.Body)
				fatal("Error %d: %s", resp.StatusCode, string(body))
			}
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				fatal("reading response: %v", err)
			}
			if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil && !os.IsExist(err) {
				fatal("creating directory: %v", err)
			}
			if err := os.WriteFile(output, data, 0644); err != nil {
				fatal("writing file: %v", err)
			}
			fmt.Printf(green("✓")+" Exported %s to "+cyan("%s")+"\n", humanBytes(int64(len(data))), output)
		},
	}
	exportCmd.Flags().StringVar(&format, "format", "json", "export format: json or csv")
	exportCmd.Flags().StringVar(&output, "output", "", "output file path")
	cmd.AddCommand(exportCmd)
	return cmd
}

func patchCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "patch", Short: "Manage virtual patches"}
	var cve string

	applyCmd := &cobra.Command{
		Use: "apply", Short: "Apply a virtual patch",
		Run: func(cmd *cobra.Command, args []string) {
			if cve == "" {
				fatal("--cve is required")
			}
			resp, err := doRequest("POST", "/patches/"+cve+"/apply", map[string]interface{}{"cve": cve})
			handleResponse(resp, err)
		},
	}
	applyCmd.Flags().StringVar(&cve, "cve", "", "CVE ID to patch")
	cmd.AddCommand(applyCmd)

	revokeCmd := &cobra.Command{
		Use: "revoke", Short: "Revoke a virtual patch",
		Run: func(cmd *cobra.Command, args []string) {
			if cve == "" {
				fatal("--cve is required")
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
	cmd := &cobra.Command{Use: "config", Short: "Manage WAF configuration"}

	validateCmd := &cobra.Command{
		Use: "validate", Short: "Validate current configuration",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := doRequest("POST", "/config/validate", nil)
			if err != nil {
				fatal("%v", err)
			}
			defer resp.Body.Close()
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				body, _ := io.ReadAll(resp.Body)
				fatal("%s", string(body))
			}
			valid, _ := result["valid"].(bool)
			if valid {
				fmt.Println(green("✓") + " " + bold("Configuration is valid"))
			} else {
				fmt.Println(red("✗") + " " + bold("Configuration is INVALID"))
				if errs, ok := result["errors"].([]interface{}); ok {
					for _, e := range errs {
						fmt.Printf("  "+red("•")+" %v\n", e)
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

	var outputFile, diffFile, applyFile string
	exportCmd := &cobra.Command{
		Use: "export", Short: "Export configuration to YAML file",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := doRequestRaw("GET", "/config/export", nil)
			if err != nil {
				fatal("%v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				body, _ := io.ReadAll(resp.Body)
				fatal("Error %d: %s", resp.StatusCode, string(body))
			}
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				fatal("reading response: %v", err)
			}
			if outputFile == "" {
				fmt.Print(string(data))
				return
			}
			if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil && !os.IsExist(err) {
				fatal("creating directory: %v", err)
			}
			if err := os.WriteFile(outputFile, data, 0644); err != nil {
				fatal("writing file: %v", err)
			}
			fmt.Printf(green("✓")+" Configuration exported to "+cyan("%s")+" (%s)\n", outputFile, humanBytes(int64(len(data))))
		},
	}
	exportCmd.Flags().StringVar(&outputFile, "output", "", "output file path (default: stdout)")
	cmd.AddCommand(exportCmd)

	diffCmd := &cobra.Command{
		Use: "diff", Short: "Diff current vs proposed config",
		Run: func(cmd *cobra.Command, args []string) {
			if diffFile == "" {
				fatal("--file is required")
			}
			data, err := os.ReadFile(diffFile)
			if err != nil {
				fatal("reading file: %v", err)
			}
			contentType := "application/json"
			if strings.HasSuffix(strings.ToLower(diffFile), ".yaml") || strings.HasSuffix(strings.ToLower(diffFile), ".yml") {
				contentType = "application/x-yaml"
			}
			req, err := newRequest(context.Background(), "POST", "/config/diff", bytes.NewReader(data))
			if err != nil {
				fatal("%v", err)
			}
			req.Header.Set("Content-Type", contentType)
			client := &http.Client{Timeout: timeout}
			resp, err := client.Do(req)
			handleResponse(resp, err)
		},
	}
	diffCmd.Flags().StringVar(&diffFile, "file", "", "path to proposed config file")
	cmd.AddCommand(diffCmd)

	applyCmd := &cobra.Command{
		Use: "apply", Short: "Apply new configuration",
		Run: func(cmd *cobra.Command, args []string) {
			if applyFile == "" {
				fatal("--file is required")
			}
			data, err := os.ReadFile(applyFile)
			if err != nil {
				fatal("reading file: %v", err)
			}
			contentType := "application/json"
			if strings.HasSuffix(strings.ToLower(applyFile), ".yaml") || strings.HasSuffix(strings.ToLower(applyFile), ".yml") {
				contentType = "application/x-yaml"
			}
			req, err := newRequest(context.Background(), "POST", "/config/import", bytes.NewReader(data))
			if err != nil {
				fatal("%v", err)
			}
			req.Header.Set("Content-Type", contentType)
			client := &http.Client{Timeout: timeout}
			resp, err := client.Do(req)
			if err != nil {
				fatal("%v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				body, _ := io.ReadAll(resp.Body)
				fatal("Error %d: %s", resp.StatusCode, string(body))
			}
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				fmt.Println(green("✓") + " " + bold("Configuration applied successfully"))
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
		Short: "Show WAF system status",
		Run: func(cmd *cobra.Command, args []string) {
			data, err := apiGet("/system/status", nil)
			if err != nil {
				// fall back to /admin/status
				resp, err := doRequest("GET", "/status", nil)
				handleResponse(resp, err)
				return
			}
			printStatus(data)
		},
	}
}

func printStatus(data map[string]interface{}) {
	status, _ := data["status"].(string)
	if status == "" {
		status = "unknown"
	}
	statusColor := green
	if status != "healthy" && status != "ok" {
		statusColor = yellow
	}

	fmt.Println()
	fmt.Printf("  "+bold("FortressWAF")+" %s\n\n", cyan(Version))

	row := func(k, v string) {
		fmt.Printf("    "+cyan("▸")+" %-18s %s\n", bold(k)+":", v)
	}

	row("Status", statusColor(status))
	if v, ok := data["version"]; ok {
		row("Version", fmt.Sprintf("%v", v))
	}
	if v, ok := data["uptime"]; ok {
		row("Uptime", fmt.Sprintf("%v", v))
	} else if v, ok := data["uptime_sec"]; ok {
		upt := int64(v.(float64))
		row("Uptime", fmtDuration(time.Duration(upt)*time.Second))
	}
	if v, ok := data["sites"]; ok {
		row("Sites", fmt.Sprintf("%v", v))
	}
	if v, ok := data["rules"]; ok {
		row("Rules", fmt.Sprintf("%v", v))
	}
	if v, ok := data["logs"]; ok {
		row("Logs", fmt.Sprintf("%v", v))
	}
	if v, ok := data["alerts"]; ok {
		row("Alerts", fmt.Sprintf("%v", v))
	}
	if v, ok := data["goroutines"]; ok {
		row("Goroutines", fmt.Sprintf("%v", v))
	}
	if v, ok := data["started_at"]; ok {
		row("Started", fmt.Sprintf("%v", v))
	}
	fmt.Println()
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show fortressctl version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf(bold("fortressctl")+" %s\n", cyan(Version))
			fmt.Printf("  "+gray("commit:")+"     %s\n", Commit)
			fmt.Printf("  "+gray("build date:")+" %s\n", BuildDate)
			fmt.Printf("  "+gray("go version:")+" %s\n", runtime.Version()[2:])
		},
	}
}

func monitorCmd() *cobra.Command {
	var window int
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Live monitoring dashboard (refresh every 3s)",
		Run: func(cmd *cobra.Command, args []string) {
			runMonitor(window)
		},
	}
	cmd.Flags().IntVar(&window, "window", 5, "monitoring window in minutes")
	return cmd
}

func runMonitor(window int) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	done := make(chan struct{})
	go func() {
		<-sig
		fmt.Print(ansiShow)
		os.Exit(0)
	}()

	fmt.Print(ansiHide)
	defer fmt.Print(ansiShow)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	// Print immediately then refresh
	printDashboard(window)
	for range ticker.C {
		printDashboard(window)
	}
	<-done
}

func printDashboard(window int) {
	var buf bytes.Buffer
	buf.WriteString(ansiClear)
	buf.WriteString(bold(" FortressWAF Live Monitor "))
	buf.WriteString(gray(fmt.Sprintf("  (window: %dm • refreshing every 3s • press Ctrl+C to quit)\n", window)))
	buf.WriteString(cyan(strings.Repeat("━", 70)) + "\n")

	// Fetch traffic + attack stats in parallel
	type trafficResult struct {
		data map[string]interface{}
		err  error
	}
	type attackResult struct {
		data map[string]interface{}
		err  error
	}
	tc := make(chan trafficResult, 1)
	ac := make(chan attackResult, 1)

	go func() {
		d, err := apiGet("/analytics/traffic", map[string]string{"window": fmt.Sprintf("%d", window)})
		tc <- trafficResult{d, err}
	}()
	go func() {
		d, err := apiGet("/analytics/attacks", map[string]string{"window": fmt.Sprintf("%d", window)})
		ac <- attackResult{d, err}
	}()

	tr := <-tc
	ar := <-ac

	if tr.err == nil && tr.data != nil {
		printTrafficPanel(&buf, tr.data)
	}
	if ar.err == nil && ar.data != nil {
		printAttackPanel(&buf, ar.data)
	}
	buf.WriteString("\n")
	os.Stdout.Write(buf.Bytes())
}

func printTrafficPanel(buf *bytes.Buffer, d map[string]interface{}) {
	toFloat := func(v interface{}) float64 {
		if v == nil {
			return 0
		}
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case int64:
			return float64(n)
		case string:
			return 0
		default:
			return 0
		}
	}

	total := toFloat(d["total_requests"])
	blocked := toFloat(d["blocked"])
	rps := toFloat(d["requests_per_sec"])
	latency := toFloat(d["avg_latency_ms"])
	bw := toFloat(d["bandwidth_bytes"])

	blockPct := 0.0
	if total > 0 {
		blockPct = blocked / total * 100
	}

	buf.WriteString(fmt.Sprintf("  %-20s %s  %-20s %s\n",
		bold("Total Requests"), humanNum(int64(total)),
		bold("Blocked"), coloredNum(int64(blocked), blockPct)))
	buf.WriteString(fmt.Sprintf("  %-20s %s  %-20s %s\n",
		bold("Requests/sec"), cyan(fmt.Sprintf("%.1f", rps)),
		bold("Block Rate"), coloredPct(blockPct)))
	buf.WriteString(fmt.Sprintf("  %-20s %s  %-20s %s\n",
		bold("Avg Latency"), cyan(fmt.Sprintf("%.1f ms", latency)),
		bold("Bandwidth"), cyan(humanBytes(int64(bw)))))

	// blocked gauge
	buf.WriteString(fmt.Sprintf("\n  %s %s %s\n",
		bold("Blocked Traffic"),
		coloredBar(blockPct, 40),
		gray(fmt.Sprintf(" %.1f%%", blockPct))))
	buf.WriteString(cyan(strings.Repeat("─", 70)) + "\n")
}

func printAttackPanel(buf *bytes.Buffer, d map[string]interface{}) {
	byType, _ := d["by_type"].(map[string]interface{})
	if byType == nil {
		return
	}

	buf.WriteString(fmt.Sprintf("  %s\n", bold("Attacks by Severity")))
	type sevEntry struct {
		sev   string
		count float64
	}
	var entries []sevEntry
	total := 0.0
	for k, v := range byType {
		c := toFloat(v)
		entries = append(entries, sevEntry{k, c})
		total += c
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})
	for _, e := range entries {
		pct := 0.0
		if total > 0 {
			pct = e.count / total * 100
		}
		c := colorForPct(pct)
		buf.WriteString(fmt.Sprintf("    "+cyan("▸")+" %-16s %s %s %s\n",
			e.sev, coloredBar(pct, 30),
			c+fmt.Sprintf("%5.1f%%", pct)+ansiReset,
			gray(fmt.Sprintf("(%s req)", humanNum(int64(e.count))))))
	}

	// by endpoint top 5
	byEndpoint, _ := d["by_endpoint"].(map[string]interface{})
	if byEndpoint != nil {
		buf.WriteString(fmt.Sprintf("\n  %s\n", bold("Top Endpoints Under Attack")))
		type epEntry struct {
			ep    string
			count float64
		}
		var eps []epEntry
		for k, v := range byEndpoint {
			eps = append(eps, epEntry{k, toFloat(v)})
		}
		sort.Slice(eps, func(i, j int) bool {
			return eps[i].count > eps[j].count
		})
		if len(eps) > 5 {
			eps = eps[:5]
		}
		for _, e := range eps {
			buf.WriteString(fmt.Sprintf("    "+gray("•")+" %-30s %s\n", e.ep, red(humanNum(int64(e.count)))))
		}
	}

	buf.WriteString(cyan(strings.Repeat("─", 70)) + "\n")
}

func statsCmd() *cobra.Command {
	var window int
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show traffic and attack statistics",
		Run: func(cmd *cobra.Command, args []string) {
			printStats(window)
		},
	}
	cmd.Flags().IntVar(&window, "window", 60, "statistics window in minutes")
	return cmd
}

func printStats(window int) {
	fmt.Println()
	fmt.Printf("  "+bold("FortressWAF Statistics")+"  "+gray("(window: %dm)\n\n"), window)

	type result struct {
		data map[string]interface{}
		err  error
		kind string
	}
	ch := make(chan result, 2)
	go func() {
		d, err := apiGet("/analytics/traffic", map[string]string{"window": fmt.Sprintf("%d", window)})
		ch <- result{d, err, "traffic"}
	}()
	go func() {
		d, err := apiGet("/analytics/attacks", map[string]string{"window": fmt.Sprintf("%d", window)})
		ch <- result{d, err, "attacks"}
	}()

	for i := 0; i < 2; i++ {
		r := <-ch
		if r.err != nil {
			fmt.Printf("  "+red("✗")+" %s: %v\n", r.kind, r.err)
			continue
		}
		switch r.kind {
		case "traffic":
			toFloat := func(v interface{}) float64 {
				if v == nil {
					return 0
				}
				switch n := v.(type) {
				case float64:
					return n
				case int:
					return float64(n)
				case int64:
					return float64(n)
				default:
					return 0
				}
			}
			total := toFloat(r.data["total_requests"])
			blocked := toFloat(r.data["blocked"])
			rps := toFloat(r.data["requests_per_sec"])
			latency := toFloat(r.data["avg_latency_ms"])
			bw := toFloat(r.data["bandwidth_bytes"])
			blockPct := 0.0
			if total > 0 {
				blockPct = blocked / total * 100
			}

			fmt.Printf("  %s\n", bold("Traffic Summary"))
			row := func(k, v string) {
				fmt.Printf("    "+cyan("▸")+" %-20s %s\n", bold(k)+":", v)
			}
			row("Total Requests", humanNum(int64(total)))
			row("Blocked", red(humanNum(int64(blocked))))
			row("Block Rate", coloredPct(blockPct))
			row("Requests/sec", cyan(fmt.Sprintf("%.1f", rps)))
			row("Avg Latency", cyan(fmt.Sprintf("%.1f ms", latency)))
			row("Bandwidth", cyan(humanBytes(int64(bw))))
			fmt.Printf("\n  %s %s\n", bold("Blocked Rate"), coloredBar(blockPct, 50))
			fmt.Println()

			// by_status
			byStatus, _ := r.data["by_status"].(map[string]interface{})
			if byStatus != nil {
				fmt.Printf("  %s\n", bold("HTTP Status Codes"))
				rows := [][]string{}
				for code, count := range byStatus {
					c := toFloat(count)
					barStr := coloredBar(c/total*100, 20)
					rows = append(rows, []string{code, fmt.Sprintf("%v", humanNum(int64(c))), barStr})
				}
				sort.Slice(rows, func(i, j int) bool { return rows[i][1] > rows[j][1] })
				printTable([]string{"Code", "Count", "Distribution"}, rows)
				fmt.Println()
			}

			// by_method
			byMethod, _ := r.data["by_method"].(map[string]interface{})
			if byMethod != nil {
				fmt.Printf("  %s\n", bold("HTTP Methods"))
				rows := [][]string{}
				for method, count := range byMethod {
					c := toFloat(count)
					rows = append(rows, []string{method, humanNum(int64(c))})
				}
				sort.Slice(rows, func(i, j int) bool { return rows[i][1] > rows[j][1] })
				printTable([]string{"Method", "Count"}, rows)
				fmt.Println()
			}

		case "attacks":
			toFloat := func(v interface{}) float64 {
				if v == nil {
					return 0
				}
				switch n := v.(type) {
				case float64:
					return n
				case int:
					return float64(n)
				case int64:
					return float64(n)
				default:
					return 0
				}
			}
			fmt.Printf("  %s\n", bold("Attack Breakdown"))
			byType, _ := r.data["by_type"].(map[string]interface{})
			if byType != nil {
				rows := [][]string{}
				total := 0.0
				for _, v := range byType {
					total += toFloat(v)
				}
				for attackType, count := range byType {
					c := toFloat(count)
					pct := 0.0
					if total > 0 {
						pct = c / total * 100
					}
					rows = append(rows, []string{attackType, humanNum(int64(c)), fmt.Sprintf("%.1f%%", pct), coloredBar(pct, 20)})
				}
				sort.Slice(rows, func(i, j int) bool { return rows[i][1] > rows[j][1] })
				printTable([]string{"Attack Type", "Count", "Pct", "Distribution"}, rows)
			}

			byCountry, _ := r.data["by_country"].(map[string]interface{})
			if byCountry != nil {
				fmt.Printf("\n  %s\n", bold("By Country"))
				rows := [][]string{}
				for country, count := range byCountry {
					rows = append(rows, []string{country, humanNum(int64(toFloat(count)))})
				}
				sort.Slice(rows, func(i, j int) bool { return rows[i][1] > rows[j][1] })
				if len(rows) > 10 {
					rows = rows[:10]
				}
				printTable([]string{"Country", "Requests"}, rows)
			}
			fmt.Println()
		}
	}
	fmt.Println()
}

func topCmd() *cobra.Command {
	var window, limit int
	cmd := &cobra.Command{
		Use:   "top",
		Short: "Show top endpoints and attackers",
		Run: func(cmd *cobra.Command, args []string) {
			printTop(window, limit)
		},
	}
	cmd.Flags().IntVar(&window, "window", 60, "window in minutes")
	cmd.Flags().IntVar(&limit, "limit", 10, "number of entries")
	return cmd
}

func printTop(window, limit int) {
	fmt.Println()
	fmt.Printf("  "+bold("FortressWAF Top Endpoints & Attackers")+"  "+gray("(window: %dm)\n\n"), window)

	ch := make(chan map[string]interface{}, 2)
	go func() {
		d, _ := apiGet("/analytics/attacks", map[string]string{"window": fmt.Sprintf("%d", window)})
		ch <- d
	}()

	d := <-ch
	toFloat := func(v interface{}) float64 {
		if v == nil {
			return 0
		}
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case int64:
			return float64(n)
		default:
			return 0
		}
	}

	byEndpoint, _ := d["by_endpoint"].(map[string]interface{})
	if byEndpoint != nil {
		fmt.Printf("  %s\n", bold("Top Endpoints"))
		type ep struct{ name string; count float64 }
		var eps []ep
		for k, v := range byEndpoint {
			eps = append(eps, ep{k, toFloat(v)})
		}
		sort.Slice(eps, func(i, j int) bool { return eps[i].count > eps[j].count })
		if len(eps) > limit {
			eps = eps[:limit]
		}
		rows := [][]string{}
		for _, e := range eps {
			rows = append(rows, []string{e.name, humanNum(int64(e.count)), coloredBar(e.count/toFloatVal(byEndpoint, eps[0].count)*100, 20)})
		}
		printTable([]string{"Endpoint", "Requests", "Distribution"}, rows)
		fmt.Println()
	}
}

func toFloatVal(m map[string]interface{}, def float64) float64 {
	return def
}

func humanNum(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.2fM", float64(n)/1000000)
}

func humanBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	if n < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB", float64(n)/(1024*1024*1024))
}

func coloredNum(n int64, pct float64) string {
	c := colorForPct(pct)
	return c + humanNum(n) + ansiReset
}

func coloredPct(pct float64) string {
	c := colorForPct(pct)
	return c + fmt.Sprintf("%.1f%%", pct) + ansiReset
}

func fmtDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, mins, secs)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, mins, secs)
	}
	if mins > 0 {
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

func toFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

var _ = io.EOF
var _ = json.Marshal
