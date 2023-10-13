locals {
  time = timestamp()
}

variable "tags" {
  type = map
  default = { "a" : "apple, "b" : "banana", "c" : "cat" }
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

output "tags" {
  value = var.tags
}
