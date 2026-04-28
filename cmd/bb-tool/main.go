package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
)

func main() {
	pingCmd := flag.NewFlagSet("ping", flag.ExitOnError)

	lsCmd := flag.NewFlagSet("ls", flag.ExitOnError)
	lsType := lsCmd.String("type", "all", "Resource type to list (traffic-config, cert, custom-page, ca-certificate, ip-sources, all)")

	rmCmd := flag.NewFlagSet("rm", flag.ExitOnError)
	rmType := rmCmd.String("type", "", "Resource type to delete (traffic-config, cert, custom-page, ca-certificate)")
	rmID := rmCmd.String("id", "", "Resource ID to delete")

	specCheckCmd := flag.NewFlagSet("spec-check", flag.ExitOnError)
	specRemoteURL := specCheckCmd.String("remote-url", "https://docs.baffinbay.com/openapi/traffic-mgmt.yml", "Remote OpenAPI URL")
	specLocalPath := specCheckCmd.String("local-path", "api-spec/traffic-mgmt-2.yml", "Local OpenAPI file path")

	if len(os.Args) < 2 {
		fmt.Println("expected 'ping', 'ls', 'rm' or 'spec-check' subcommands")
		os.Exit(1)
	}

	c := initClient()

	ctx := context.Background()

	switch os.Args[1] {
	case "ping":
		_ = pingCmd.Parse(os.Args[2:])
		err := c.Ping(ctx)
		if err != nil {
			fmt.Printf("Ping failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Ping successful!")

	case "ls":
		_ = lsCmd.Parse(os.Args[2:])
		handleLs(ctx, c, *lsType)

	case "rm":
		_ = rmCmd.Parse(os.Args[2:])
		handleRm(ctx, c, *rmType, *rmID)

	case "spec-check":
		_ = specCheckCmd.Parse(os.Args[2:])
		handleSpecCheck(*specRemoteURL, *specLocalPath)

	default:
		fmt.Println("expected 'ping', 'ls', 'rm' or 'spec-check' subcommands")
		os.Exit(1)
	}
}

func initClient() *client.Client {
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

	if clientID == "" || clientSecret == "" {
		fmt.Println("BAFFINBAY_CLIENT_ID and BAFFINBAY_CLIENT_SECRET must be set")
		os.Exit(1)
	}

	ts, err := client.BuildTokenSource(context.Background(), client.TokenSourceConfig{
		OIDCURL:      oidcURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		fmt.Printf("Token source initialization failed: %v\n", err)
		os.Exit(1)
	}

	c := client.NewClient(apiURL)
	c.AccountID = accountID
	c.TokenSource = ts
	return c
}

func handleLs(ctx context.Context, c *client.Client, resourceType string) {
	if resourceType == "all" || resourceType == "traffic-config" {
		fmt.Println("--- Traffic Configurations ---")
		resp, err := c.GetTrafficConfigs(ctx)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			for _, tc := range resp.Data {
				fmt.Printf("ID: %s, Name: %s\n", tc.ID, tc.Attributes.Name)
			}
		}
	}

	if resourceType == "all" || resourceType == "cert" {
		fmt.Println("\n--- Certificates ---")
		resp, err := c.GetCertificates(ctx)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			for _, cert := range resp.Data {
				fmt.Printf("ID: %s, Common Name: %s\n", cert.ID, cert.Attributes.CommonName)
			}
		}
	}

	if resourceType == "all" || resourceType == "custom-page" {
		fmt.Println("\n--- Custom Pages ---")
		resp, err := c.GetCustomPages(ctx, c.AccountID)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			for _, cp := range resp.Data {
				fmt.Printf("ID: %s, Name: %s\n", cp.ID, cp.Attributes.Name)
			}
		}
	}

	if resourceType == "all" || resourceType == "ca-certificate" {
		fmt.Println("\n--- CA Certificates ---")
		resp, err := c.GetCaCertificates(ctx)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			for _, ca := range resp.Data {
				fmt.Printf("ID: %s, Name: %s\n", ca.ID, ca.Attributes.Name)
			}
		}
	}

	if resourceType == "all" || resourceType == "ip-sources" {
		fmt.Println("\n--- IP Sources ---")
		resp, err := c.GetIpSources(ctx)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			for _, ip := range resp.Data {
				fmt.Printf("ID: %s, CIDR: %s\n", ip.ID, ip.Attributes.CIDR)
			}
		}
	}
}

func handleRm(ctx context.Context, c *client.Client, resourceType, id string) {
	if resourceType == "" || id == "" {
		fmt.Println("type and id are required for rm")
		os.Exit(1)
	}

	var err error
	switch resourceType {
	case "traffic-config":
		err = c.DeleteTrafficConfig(ctx, id)
	case "cert":
		err = c.DeleteCertificate(ctx, id)
	case "custom-page":
		err = c.DeleteCustomPage(ctx, id)
	case "ca-certificate":
		err = c.DeleteCaCertificate(ctx, id)
	default:
		fmt.Printf("Invalid resource type: %s\n", resourceType)
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Failed to delete %s %s: %v\n", resourceType, id, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully deleted %s %s\n", resourceType, id)
}
