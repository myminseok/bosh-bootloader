variable "system_domain" {
  type = "string"
}

variable "pfx_cert_base64" {
  type = "string"
}

variable "pfx_password" {
  type = "string"
}

# Create a application gateway in the web_servers resource group
resource "azurerm_virtual_network" "cf-vn" {
  name                = "${var.env_id}-cf-vnet"
  resource_group_name = "${azurerm_resource_group.bosh.name}"
  address_space       = ["${var.network_cidr}"]
  location            = "${var.region}"
}

resource "azurerm_subnet" "sub1" {
  name                 = "${var.env_id}-cf-subnet1"
  resource_group_name  = "${azurerm_resource_group.bosh.name}"
  virtual_network_name = "${azurerm_virtual_network.cf-vn.name}"
  address_prefix       = "${var.internal_cidr}"
}

resource "azurerm_public_ip" "lb" {
  name                         = "${var.env_id}-cf-lb-ip"
  location                     = "${var.region}"
  resource_group_name          = "${azurerm_resource_group.bosh.name}"
  public_ip_address_allocation = "dynamic"
}

# Create an application gateway
resource "azurerm_application_gateway" "network" {
  name                = "${var.env_id}-app-gateway"
  resource_group_name = "${azurerm_resource_group.bosh.name}"
  location            = "West US"
 
  sku {
    name           = "Standard_Small"
    tier           = "Standard"
    capacity       = 2
  }

  probe {
    name = "Probe01"
    protocol = "Http"
    path = "/login"
    host = "login.${var.system_domain}"
    interval = 60
    timeout = 60
    unhealthy_threshold = 3
  }
 
  gateway_ip_configuration {
    name         = "${var.env_id}-cf-gateway-ip-configuration"
    subnet_id    = "${azurerm_virtual_network.cf-vn.id}/subnets/${azurerm_subnet.sub1.name}"
  }
 
  frontend_port {
    name         = "frontendporthttps"
    port         = 443
  }

  frontend_port {
    name         = "frontendportlogs"
    port         = 4443
  }
 
  frontend_ip_configuration {
    name         = "${var.env_id}-cf-frontend-ip-configuration"
    public_ip_address_id = "${azurerm_public_ip.lb.id}"
  }

  backend_address_pool {
    name = "${var.env_id}-cf-backend-address-pool"
  }
 
  backend_http_settings {
    name                  = "${azurerm_virtual_network.cf-vn.name}-be-htst"
    cookie_based_affinity = "Disabled"
    port                  = 80
    protocol              = "Http"
    request_timeout       = 1
    probe_name            = "Probe01"
  }

  ssl_certificate {
    name = "ssl-cert"
    data = "${var.pfx_cert_base64}"
    password = "${var.pfx_password}"
  }
 
  http_listener {
    name                                  = "${azurerm_virtual_network.cf-vn.name}-httplstn"
    frontend_ip_configuration_name        = "${var.env_id}-cf-frontend-ip-configuration"
    frontend_port_name                    = "frontendporthttps"
    protocol                              = "Https"
    ssl_certificate_name                  = "ssl-cert"
  }
 
  request_routing_rule {
    name                       = "${azurerm_virtual_network.cf-vn.name}-rqrt"
    rule_type                  = "Basic"
    http_listener_name         = "${azurerm_virtual_network.cf-vn.name}-httplstn"
    backend_address_pool_name  = "${var.env_id}-cf-backend-address-pool"
    backend_http_settings_name = "${azurerm_virtual_network.cf-vn.name}-be-htst"
  }
}

output "app_gateway_name" {
	value = "${azurerm_application_gateway.network.name}"
}