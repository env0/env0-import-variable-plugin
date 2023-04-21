resource "null_resource" "null" {
}

output "test_envid" {
  value = var.test_envid
}

output "test_envname" {
  value = var.test_envname
}

variable "test_envid" {
  type = string
}

variable "test_envname" {
  type = string
}