output "time" {
  value = timestamp()
}

output "time_json" {
  value = jsonencode({ "time" : output.time.value })
}

output "time_multiline" {
  value = "This last ran on:\r\n${output.time.value}"
}
