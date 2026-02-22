package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
)

func main() {
	pingCmd := flag.NewFlagSet("ping", flag.ExitOnError)

	lsCmd := flag.NewFlagSet("ls", flag.ExitOnError)
	lsType := lsCmd.String("type", "all", "Resource type to list (traffic-config, cert, custom-page, ca-certificate, ip-sources, all)")

	rmCmd := flag.NewFlagSet("rm", flag.ExitOnError)
	rmType := rmCmd.String("type", "", "Resource type to delete (traffic-config, cert, custom-page, ca-certificate)")
	rmID := rmCmd.String("id", "", "Resource ID to delete")

	if len(os.Args) < 2 {
		fmt.Println("expected 'ping', 'ls' or 'rm' subcommands")
		os.Exit(1)
	}

	c := initClient()

	switch os.Args[1] {
	case "ping":
		_ = pingCmd.Parse(os.Args[2:])
		err := c.Ping()
		if err != nil {
			fmt.Printf("Ping failed: %v\\n", err)
			os.Exit(1)
		}
		fmt.Println("Ping successful!")

	case "ls":
		_ = lsCmd.Parse(os.Args[2:])
		handleLs(c, *lsType)

	case "rm":
		_ = rmCmd.Parse(os.Args[2:])
		handleRm(c, *rmType, *rmID)

	default:
		fmt.Println("expected 'ping', 'ls' or 'rm' subcommands")
		os.Exit(1)
	}
}

func initClient() *client.Client {
	// Manually parse .env since we don't have godotenv
	if file, err := os.Open(".env"); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				// Remove quotes if present
				val = strings.Trim(val, `"'`)
				os.Setenv(key, val)
			}
		}
		_ = file.Close()
	}

	apiURL := os.Getenv("BAFFINBAY_API_URL")
	if apiURL == "" {
		apiURL = "https://portal.baffinbay.com"
	}
	oidcURL := os.Getenv("BAFFINBAY_OIDC_URL")
	if oidcURL == "" {
		oidcURL = "https://m2m-auth.baffinbay.com/oauth/token"
	}
	clientID := os.Getenv("BAFFINBAY_CLIENT_ID")
	clientSecret := os.Getenv("BAFFINBAY_CLIENT_SECRET")
	accountID := os.Getenv("BAFFINBAY_ACCOUNT_ID")
	apiKey := os.Getenv("BAFFINBAY_API_KEY")

	c := client.NewClient(apiURL)
	c.AccountID = accountID

	if apiKey != "" {
		c.Token = apiKey
		// Check if current token is valid
		if err := c.Ping(); err == nil {
			return c
		}
		fmt.Println("Existing API key invalid, attempting to refresh...")
	}

	if clientID != "" && clientSecret != "" {
		// Force refresh if the key we had didn't work
		err := c.Authenticate(oidcURL, clientID, clientSecret, true)
		if err != nil {
			fmt.Printf("Authentication failed: %v\\n", err)
			os.Exit(1)
		}
	} else if c.Token == "" {
		fmt.Println("Either BAFFINBAY_API_KEY or (BAFFINBAY_CLIENT_ID and BAFFINBAY_CLIENT_SECRET) must be set")
		os.Exit(1)
	}

	return c
}

func handleLs(c *client.Client, resourceType string) {
	if resourceType == "all" || resourceType == "traffic-config" {
		fmt.Println("--- Traffic Configurations ---")
		resp, err := c.GetTrafficConfigs()
		if err != nil {
			fmt.Printf("Error: %v\\n", err)
		} else {
			for _, tc := range resp.Data {
				fmt.Printf("ID: %s, Name: %s\\n", tc.ID, tc.Attributes.Name)
			}
		}
	}

	if resourceType == "all" || resourceType == "cert" {
		fmt.Println("\\n--- Certificates ---")
		resp, err := c.GetCertificates()
		if err != nil {
			fmt.Printf("Error: %v\\n", err)
		} else {
			for _, cert := range resp.Data {
				fmt.Printf("ID: %s, Common Name: %s\\n", cert.ID, cert.Attributes.CommonName)
			}
		}
	}

	if resourceType == "all" || resourceType == "custom-page" {
		fmt.Println("\\n--- Custom Pages ---")
		resp, err := c.GetCustomPages(c.AccountID)
		if err != nil {
			fmt.Printf("Error: %v\\n", err)
		} else {
			for _, cp := range resp.Data {
				fmt.Printf("ID: %s, Name: %s\\n", cp.ID, cp.Attributes.Name)
			}
		}
	}

	if resourceType == "all" || resourceType == "ca-certificate" {
		fmt.Println("\\n--- CA Certificates ---")
		resp, err := c.GetCaCertificates()
		if err != nil {
			fmt.Printf("Error: %v\\n", err)
		} else {
			for _, ca := range resp.Data {
				fmt.Printf("ID: %s, Name: %s\\n", ca.ID, ca.Attributes.Name)
			}
		}
	}

	if resourceType == "all" || resourceType == "ip-sources" {
		fmt.Println("\\n--- IP Sources ---")
		resp, err := c.GetIpSources()
		if err != nil {
			fmt.Printf("Error: %v\\n", err)
		} else {
			for _, ip := range resp.Data {
				fmt.Printf("ID: %s, CIDR: %s\\n", ip.ID, ip.Attributes.CIDR)
			}
		}
	}
}

func handleRm(c *client.Client, resourceType, id string) {
	if resourceType == "" || id == "" {
		fmt.Println("type and id are required for rm")
		os.Exit(1)
	}

	var err error
	switch resourceType {
	case "traffic-config":
		err = c.DeleteTrafficConfig(id)
	case "cert":
		err = c.DeleteCertificate(id)
	case "custom-page":
		err = c.DeleteCustomPage(id)
	case "ca-certificate":
		err = c.DeleteCaCertificate(id)
	default:
		fmt.Printf("Invalid resource type: %s\\n", resourceType)
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Failed to delete %s %s: %v\\n", resourceType, id, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully deleted %s %s\\n", resourceType, id)
}
