terraform {
  required_providers {
    baffinbay = {
      source = "baffinbay/baffinbay"
    }
  }
}

provider "baffinbay" {
  # Credentials are read from BAFFINBAY_CLIENT_ID / BAFFINBAY_CLIENT_SECRET
  # environment variables by default.
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
