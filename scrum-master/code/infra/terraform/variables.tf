variable "project" {
  type    = string
  default = "scrum-master-agent"
}

variable "environment" {
  type    = string
  default = "dev"
}

variable "location" {
  type    = string
  default = "eastus"
}

variable "tags" {
  type = map(string)
  default = {
    project = "scrum-master-agent"
    owner   = "aaraminds"
  }
}
