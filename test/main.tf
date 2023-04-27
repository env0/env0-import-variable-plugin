resource "null_resource" "null" {
}

output "test_envid" {
  value = var.test_envid
}

output "test_envname" {
  value = var.test_envname
}

output "test_json" {
  value = var.test_json
}

output "test_multline" {
  value = var.test_multline
}

variable "test_envid" {
  type = string
}

variable "test_envname" {
  type = string
}

variable "test_json" {
  type = object
}

variable "test_multline" {
  type = string
}