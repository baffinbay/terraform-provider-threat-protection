data "baffinbay_ip_lists" "all" {}

output "ip_lists" {
  value = data.baffinbay_ip_lists.all.ip_lists
}
