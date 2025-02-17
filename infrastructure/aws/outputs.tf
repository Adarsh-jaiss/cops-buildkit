output "kms_account_key_arn" {
  value = module.base.kms_account_key_arn
}

output "kms_account_key_id" {
  value = module.base.kms_account_key_id
}

output "vpc_id" {
  value = module.base.vpc_id
}

output "private_subnet_ids" {
  value = module.base.private_subnet_ids
}

output "public_subnets_ids" {
  value = module.base.public_subnets_ids
}

output "s3_log_bucket_name" {
  value = module.base.s3_log_bucket_name
}

output "public_nat_ips" {
  value = module.base.public_nat_ips
}


output "k8s_endpoint" {
  value = module.k8scluster.k8s_endpoint
}

output "k8s_ca_data" {
  value = module.k8scluster.k8s_ca_data
}

output "k8s_cluster_name" {
  value = module.k8scluster.k8s_cluster_name
}

output "k8s_openid_provider_url" {
  value = module.k8scluster.k8s_openid_provider_url
}

output "k8s_openid_provider_arn" {
  value = module.k8scluster.k8s_openid_provider_arn
}

output "k8s_node_group_security_id" {
  value = module.k8scluster.k8s_node_group_security_id
}

output "k8s_version" {
  value = module.k8scluster.k8s_version
}


output "providers" {
  value = {
    aws = {
      region     = "ap-south-1"
      account_id = "609973658768"
    }
  }
}

output "state_storage" {
  value = "cops-production-ap-south-1"
}