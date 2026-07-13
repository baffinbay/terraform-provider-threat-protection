resource "baffinbay_ip_list" "office_networks" {
  name = "office-networks"

  entries = [
    {
      value = "198.51.100.0/24"
      note  = "Stockholm office"
    },
    {
      value = "2001:db8:100::/48"
      note  = "Office IPv6"
    }
  ]
}
