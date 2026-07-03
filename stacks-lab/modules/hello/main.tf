variable "name" {
  type        = string
  description = "Name to embed in the terraform_data trigger."
}

resource "terraform_data" "hello" {
  input = "hello, ${var.name}"
}

output "greeting" {
  value = terraform_data.hello.output
}
