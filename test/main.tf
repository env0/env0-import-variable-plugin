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

output "test_multiline" {
  value = var.test_multiline
}

output "test_workflow" {
  value = var.test_workflow
}

output "test_workflow_json" {
  value = var.test_workflow_json
}

variable "test_envid" {
  type = string
}

variable "test_envname" {
  type = string
}

variable "test_json" {
  type = map
}

variable "test_multiline" {
  type = string
}

variable "test_workflow" {
  type = string
}

variable "test_workflow_json" {
  type = map
}