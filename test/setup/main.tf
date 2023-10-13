locals {
  time = timestamp()
}

output "time" {
  value = local.time
}

output "time_json" {
  value = jsonencode({ "time" : local.time, "a" : "apple", "b" : "banana" })
}

output "test_multiline" {
  value = "This last ran on:\r\n${local.time}"
}
