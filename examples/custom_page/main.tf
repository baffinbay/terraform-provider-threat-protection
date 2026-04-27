terraform {
  required_providers {
    baffinbay = {
      source = "baffinbay/baffinbay"
    }
  }
}

provider "baffinbay" {
  # API Key/Token will be read from BAFFINBAY_API_KEY environment variable
  # or you can uncomment and set it here:
  # api_key = "your-bearer-token"
}

resource "baffinbay_custom_page" "example" {
  name    = "example.html"
  content = <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>Example Custom Page</title>
</head>
<body>
    <h1>Hello from Terraform!</h1>
    <p>This custom page was created via the Baffin Bay Terraform Provider.</p>
</body>
</html>
EOF
}
