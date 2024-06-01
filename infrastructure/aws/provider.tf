provider "helm" {
  kubernetes {
    host                   = module.k8scluster.k8s_endpoint
    token                  = data.aws_eks_cluster_auth.k8s.token
    cluster_ca_certificate = base64decode(module.k8scluster.k8s_ca_data)
  }
}
provider "aws" {
  region = "ap-south-1"

  allowed_account_ids = ["XXXXXX"]
}