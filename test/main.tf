resource "null_resource" "null" {
}

output "test_envid" {
  value = var.test_input
}

output "test_envname" {
  value = var.test_input
}

variable "test_envid" {
  type = string
}

variable "test_envname" {
  type = string
}