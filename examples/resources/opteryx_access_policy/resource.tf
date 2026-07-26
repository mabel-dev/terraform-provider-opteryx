resource "opteryx_access_policy" "analytics_reader" {
  workspace = "analytics"
  principal = "jane@example.com"
  role      = "reader"
  pattern   = "analytics.sales.*"
}

# A wildcard-principal grant must name an exact resource -- policy.opteryx
# rejects a pattern with glob characters when the principal is "*".
resource "opteryx_access_policy" "public_dashboard" {
  workspace = "analytics"
  principal = "*"
  role      = "reader"
  pattern   = "analytics.dashboards.q3_summary"
}
