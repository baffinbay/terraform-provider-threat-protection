package provider

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type httpProxyLiveAcceptanceConfig struct {
	Name            string
	Host            string
	BotHost         string
	FrontendIPv4    string
	BackendAddress  string
	BackendPort     int
	DeploymentState string
	CertificatePEM  string
	KeyPEM          string
	CACertPEM       string
}

func TestAccHTTPProxyResourceFullLive(t *testing.T) {
	cfg := newHTTPProxyLiveAcceptanceConfig(t)

	t.Setenv("BAFFINBAY_TOKEN_CACHE", t.TempDir()+"/token-cache.json")

	tfresource.Test(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: httpProxyLiveAcceptanceHCL(cfg, false),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttrSet("baffinbay_http_proxy.full", "id"),
					tfresource.TestCheckResourceAttr("baffinbay_http_proxy.full", "name", cfg.Name),
					tfresource.TestCheckResourceAttr("baffinbay_http_proxy.full", "deployment_state", "UNDEPLOYED"),
				),
			},
			{
				Config: httpProxyLiveAcceptanceHCL(cfg, true),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("baffinbay_http_proxy.full", "name", cfg.Name+"-updated"),
					tfresource.TestCheckResourceAttr("baffinbay_http_proxy.full", "traffic_rules.0.actions.bot_protection.strategy", "DISABLED"),
					tfresource.TestCheckResourceAttr("baffinbay_http_proxy.full", "traffic_rules.1.actions.redirect.status_code", "302"),
				),
			},
		},
	})
}

func newHTTPProxyLiveAcceptanceConfig(t *testing.T) httpProxyLiveAcceptanceConfig {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Skipping live HTTP proxy acceptance test. Set TF_ACC=1 to run.")
	}
	for _, name := range []string{"BAFFINBAY_CLIENT_ID", "BAFFINBAY_CLIENT_SECRET", "BAFFINBAY_TENANT_ID"} {
		if os.Getenv(name) == "" {
			t.Skipf("Skipping live HTTP proxy acceptance test. Set %s to run.", name)
		}
	}

	frontendIPv4 := os.Getenv("BAFFINBAY_ACC_FRONTEND_IPV4")
	if frontendIPv4 == "" {
		t.Skip("Skipping live HTTP proxy acceptance test. Set BAFFINBAY_ACC_FRONTEND_IPV4 to a non-production test IP.")
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	host := envOrDefault("BAFFINBAY_ACC_HOST", "tf-acc-"+suffix+".example.com")
	botHost := envOrDefault("BAFFINBAY_ACC_BOT_HOST", "bots-"+host)
	certPEM, keyPEM := generateAcceptanceCertificate(t, []string{host, botHost}, false)
	caCertPEM, _ := generateAcceptanceCertificate(t, []string{"tf-acc-client-ca-" + suffix + ".example.com"}, true)

	return httpProxyLiveAcceptanceConfig{
		Name:            "tf-acc-http-proxy-" + suffix,
		Host:            host,
		BotHost:         botHost,
		FrontendIPv4:    frontendIPv4,
		BackendAddress:  envOrDefault("BAFFINBAY_ACC_BACKEND_ADDRESS", "origin.example.com"),
		BackendPort:     8443,
		DeploymentState: envOrDefault("BAFFINBAY_ACC_DEPLOYMENT_STATE", "UNDEPLOYED"),
		CertificatePEM:  certPEM,
		KeyPEM:          keyPEM,
		CACertPEM:       caCertPEM,
	}
}

func generateAcceptanceCertificate(t *testing.T, dnsNames []string, isCA bool) (string, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("generate certificate serial: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: dnsNames[0],
		},
		DNSNames:              dnsNames,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}
	if isCA {
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	return string(certPEM), string(keyPEM)
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func httpProxyLiveAcceptanceHCL(cfg httpProxyLiveAcceptanceConfig, full bool) string {
	proxyName := cfg.Name
	deploymentState := "UNDEPLOYED"
	body := httpProxyLiveMinimalBody(cfg)
	if full {
		proxyName = cfg.Name + "-updated"
		deploymentState = cfg.DeploymentState
		body = httpProxyLiveFullBody(cfg)
	}

	return fmt.Sprintf(`
provider "baffinbay" {}

resource "baffinbay_certificate" "server" {
  type = "pem"
  certificate = <<EOT
%s
EOT
  key = <<EOT
%s
EOT
}

resource "baffinbay_ca_certificate" "client_ca" {
  certificate = <<EOT
%s
EOT
}

resource "baffinbay_custom_page" "rate_limit" {
  name    = "%s-429.html"
  content = "<html><body>rate limited</body></html>"
}

resource "baffinbay_custom_page" "internal_error" {
  name    = "%s-500.html"
  content = "<html><body>internal error</body></html>"
}

resource "baffinbay_custom_page" "network_error" {
  name    = "%s-502.html"
  content = "<html><body>network error</body></html>"
}

resource "baffinbay_ip_list" "blocked" {
  name = "%s-blocked"
  entries = [
    { value = "198.51.100.0/24", note = "tf-acc" },
  ]
}

resource "baffinbay_http_proxy" "full" {
  name                     = %q
  deployment_state         = %q
  connection_reuse_enabled = false

%s
}
`, cfg.CertificatePEM, cfg.KeyPEM, cfg.CACertPEM, cfg.Name, cfg.Name, cfg.Name, cfg.Name, proxyName, deploymentState, body)
}

func httpProxyLiveMinimalBody(cfg httpProxyLiveAcceptanceConfig) string {
	return fmt.Sprintf(`
  frontend = {
    connection_type = "SECURE"
    ipv4            = %q
    port            = 5991
    redirect_http   = false
    hosts = [
      {
        host           = %q
        certificate_id = baffinbay_certificate.server.id
        tls_config     = "INTERMEDIATE"
      },
    ]
  }

  backend = {
    hosts = [
      {
        address = %q
        port    = %d
      },
    ]
    server_name     = %q
    delivery_method = "LEAST_CONNECTIONS"
    tls_settings = {
      verify_certificate = {
        mode = "DISABLED"
      }
    }
  }

  protocol_settings = {
    version           = "HTTP1.1"
    enable_websockets = true
  }
`, cfg.FrontendIPv4, cfg.Host, cfg.BackendAddress, cfg.BackendPort, cfg.BackendAddress)
}

func httpProxyLiveFullBody(cfg httpProxyLiveAcceptanceConfig) string {
	return fmt.Sprintf(`
  data_protection = {
    log_redaction = {
      headers = ["Authorization"]
      cookies = ["session_id", "aaa"]
    }
  }

  frontend = {
    connection_type = "SECURE"
    ipv4            = %q
    port            = 5991
    redirect_http   = false
    hsts = {
      enabled            = true
      max_age            = 87654444
      include_subdomains = false
      preload            = false
    }
    hosts = [
      {
        host           = %q
        certificate_id = baffinbay_certificate.server.id
        tls_config     = "INTERMEDIATE"
      },
      {
        host           = %q
        certificate_id = baffinbay_certificate.server.id
        tls_config     = "INTERMEDIATE"
      },
    ]
    client_certificate_verification = {
      mode               = "VERIFY_AND_REJECT"
      ca_certificate_ids = [baffinbay_ca_certificate.client_ca.id]
    }
  }

  backend = {
    hosts = [
      {
        address = %q
        port    = %d
      },
    ]
    server_name     = %q
    delivery_method = "LEAST_CONNECTIONS"
    tls_settings = {
      client_certificate_id = baffinbay_certificate.server.id
      verify_certificate = {
        mode = "DISABLED"
      }
    }
  }

  custom_pages = [
    { id = baffinbay_custom_page.rate_limit.id, type = "429_RATE_LIMIT_BLOCK" },
    { id = baffinbay_custom_page.internal_error.id, type = "500_INTERNAL_ERROR" },
    { id = baffinbay_custom_page.network_error.id, type = "502_NETWORK_ERROR" },
  ]

  bot_protection = {
    strategy       = "ALWAYS_ON"
    challenge_type = "JS"
  }

  protocol_settings = {
    version           = "HTTP1.1"
    enable_websockets = true
  }

  waf = {
    enforcement           = "BLOCK"
    paranoia_level        = 1
    core_rule_set_version = "4.12.0"
    matched_data_enabled  = true
    source_exclusions = {
      enabled = true
      sources = ["198.51.100.8/32", "203.0.113.0/24"]
    }
    http_compliance = {
      parameter_limit = {
        enabled = true
        limit   = 502
      }
      allowed_methods = [
        "GET",
        "POST",
        "PATCH",
        "PUT",
        "TRACK",
        "X-MS-ENUMATTS",
        "REPORT",
        "RPC_IN_DATA",
        "PROPPATCH",
        "PROPFIND",
        "POLL",
        "OPTIONS",
        "NOTIFY",
        "MOVE",
        "MKWORKSPACE",
        "MKCOL",
        "MERGE",
        "LOCK",
        "LINK",
        "HEAD",
        "DELETE",
        "CONNECT",
        "CHECKOUT",
        "CHECKIN",
        "BPROPPATCH",
        "BPROPFIND",
        "BMOVE",
        "BDELETE",
        "BCOPY",
        "ACL",
        "RPC_OUT_DATA",
        "SEARCH",
        "SUBSCRIBE",
        "TRACE",
        "UNLINK",
        "UNLOCK",
        "UNSUBSCRIBE",
        "VERSION_CONTROL",
      ]
      allowed_versions = ["HTTP/1.1", "HTTP/2.0", "HTTP/2"]
      resource_configs = [
        {
          matches                         = ["/here"]
          parse_json_enabled              = true
          parse_xml_enabled               = false
          parse_multipart_request_enabled = true
        },
        {
          matches                         = ["/gurrr"]
          parse_json_enabled              = false
          parse_xml_enabled               = true
          parse_multipart_request_enabled = true
        },
        {
          matches                         = ["/aaaa", "/root"]
          parse_json_enabled              = false
          parse_xml_enabled               = false
          parse_multipart_request_enabled = false
        },
        {
          matches                         = ["/{*}"]
          parse_json_enabled              = false
          parse_xml_enabled               = true
          parse_multipart_request_enabled = false
        },
      ]
    }
    path_exclusions = [
      { match = "/this-was-fun", disable_all = false, rule_ids = [] },
      { match = "/this-was-fun{*}", disable_all = true, rule_ids = [] },
      { match = "/here/we/go/again", disable_all = true, rule_ids = [] },
    ]
    cookie_exclusions = [
      { cookie_name = "session_id", exclude_all_rules = false, rule_ids = [] },
      { cookie_name = "Authorization", exclude_all_rules = false, rule_ids = [] },
    ]
    staged_waf = {
      state                 = "ENABLED"
      mode                  = "CRS_VERSION"
      core_rule_set_version = "4.*.*"
      cookie_exclusions = [
        { cookie_name = "session_id", exclude_all_rules = false, rule_ids = [] },
        { cookie_name = "Authorization", exclude_all_rules = false, rule_ids = [] },
      ]
      path_exclusions = [
        { match = "/this-was-fun", disable_all = false, rule_ids = [] },
        { match = "/this-was-fun{*}", disable_all = true, rule_ids = [] },
      ]
    }
  }

  ip_based_access_control = {
    default_policy = "ALLOW"
    rules = {
      ip_ranges = [
        {
          address               = "198.51.100.10/32"
          policy                = "ALLOW"
          bypass_bot_protection = false
        },
        {
          address               = "2001:db8::/48"
          policy                = "ALLOW"
          bypass_bot_protection = true
        },
      ]
      asns = [
        {
          asn    = 13335
          policy = "BLOCK"
          note   = "tf-acc"
        },
      ]
      ip_lists = [
        {
          id                    = baffinbay_ip_list.blocked.id
          policy                = "BLOCK"
          bypass_bot_protection = true
        },
      ]
      geo_locations = [
        {
          region = "DZ"
          policy = "BLOCK"
          note   = "Only bad traffic"
        },
      ]
    }
  }

  rate_limiting = {
    by_source_ip = {
      enforcement = "BLOCK"
      rate = {
        value = 50
        unit  = "r/m"
      }
      burst = 200
    }
    by_source_ip_and_url = {
      enforcement = "BLOCK"
      rate = {
        value = 25
        unit  = "r/m"
      }
      burst = 150
    }
    exclusions = ["198.51.100.0/24", "203.0.113.4/31", "2001:db8::/48"]
  }

  traffic_rules = [
    {
      name = "disable bot"
      matching_conditions = [
        {
          paths = ["/bots-allowed"]
          hosts = {
            type = "ALL"
          }
        },
      ]
      actions = {
        bot_protection = {
          strategy = "DISABLED"
        }
      }
    },
    {
      name = "the other one"
      matching_conditions = [
        {
          paths = ["/root", "/definitely-no-bots{*}"]
          hosts = {
            type   = "SELECT"
            values = [%q, %q]
          }
        },
      ]
      actions = {
        redirect = {
          status_code          = 302
          url                  = "https://www.google.com"
          append_original_path = true
        }
      }
    },
    {
      name = "full action shape"
      matching_conditions = [
        {
          paths = ["/full-action{*}"]
          hosts = {
            type = "ALL"
          }
        },
      ]
      actions = {
        backends = [
          {
            address = %q
            port    = %d
          },
        ]
        headers     = [{ key = "X-Proxy", value = "baffinbay" }]
        host_header = %q
        rate_limit = {
          by_source_ip = {
            enforcement = "BLOCK"
            rate = {
              value = 10
              unit  = "r/s"
            }
            burst = 100
          }
          by_source_ip_and_url = {
            enforcement = "BLOCK"
            rate = {
              value = 10
              unit  = "r/s"
            }
            burst = 100
          }
        }
        max_body_size = {
          enforcement = "ENABLED"
          value_bytes = 1048576
        }
      }
    },
  ]
`, cfg.FrontendIPv4, cfg.Host, cfg.BotHost, cfg.BackendAddress, cfg.BackendPort, cfg.BackendAddress, cfg.Host, cfg.BotHost, cfg.BackendAddress, cfg.BackendPort, cfg.BackendAddress)
}
