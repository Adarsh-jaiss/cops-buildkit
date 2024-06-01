
module "base" {
  source                   = "yindia/saas/opta//modules/aws_base"
  version                  = "0.0.2-beta2"
  env_name                 = "production"
  layer_name               = "production"
  module_name              = "base"
  total_ipv4_cidr_block    = "10.0.0.0/16"
  vpc_log_retention        = 90
  private_ipv4_cidr_blocks = ["10.0.128.0/21", "10.0.136.0/21", "10.0.144.0/21"]
  public_ipv4_cidr_blocks  = ["10.0.0.0/21", "10.0.8.0/21", "10.0.16.0/21"]
}