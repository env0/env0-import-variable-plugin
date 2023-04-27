resource "null_resource" "null" {
}

output "test_envid" {
  value = var.test_envid
}

output "test_envname" {
  value = var.test_envname
}

output "test_json" {
  value = jsonencode({"time"=var.test_envname})
}

output "test_multiline" {
  value = "ab\r\ncd\r\n\ef"
}

variable "test_envid" {
  type = string
}

variable "test_envname" {
  type = string
}

