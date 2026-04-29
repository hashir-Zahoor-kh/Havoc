# EKS addons + the IRSA plumbing they need.
#
# Modern EKS no longer ships the EBS CSI driver in the cluster image;
# it's a managed addon you opt into. Without it, every PVC sits in
# Pending forever because no provisioner answers
# `ebs.csi.aws.com`. Strimzi's broker PVC was the first thing that
# revealed this — anything that uses persistent storage on this
# cluster will need the addon installed.
#
# Why a managed addon (vs `helm install aws-ebs-csi-driver`):
# - The addon is the AWS-supported path, version-pinned to known-good
#   builds for the cluster's k8s version.
# - Upgrades are atomic — bumping the addon version drains and
#   redeploys cleanly without us writing reconciliation logic.
# - `kubectl get addons` is a stable surface for SREs to check from.

# ---------- OIDC provider (IRSA prerequisite) ----------
#
# EKS publishes an OIDC issuer when the cluster comes up but does
# *not* automatically register it as an IAM identity provider. IRSA
# (IAM Roles for ServiceAccounts) requires the registration so STS
# can verify the JWT a pod presents when it calls AssumeRoleWithWebIdentity.
#
# The thumbprint pin is required by AWS for federated providers; we
# read the live cert via the tls provider so the value updates
# automatically if AWS rotates the OIDC signing cert.

data "tls_certificate" "eks_oidc" {
  url = aws_eks_cluster.this.identity[0].oidc[0].issuer
}

resource "aws_iam_openid_connect_provider" "eks" {
  url             = aws_eks_cluster.this.identity[0].oidc[0].issuer
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [data.tls_certificate.eks_oidc.certificates[0].sha1_fingerprint]
}

# ---------- EBS CSI driver IAM role ----------
#
# Trust policy: only the kube-system/ebs-csi-controller-sa
# ServiceAccount in this specific cluster's OIDC provider may assume
# the role. The two StringEquals conditions are belt-and-suspenders —
# `sub` pins the ServiceAccount, `aud` pins the audience to STS so a
# token minted for some other audience can't be replayed here.

data "aws_iam_policy_document" "ebs_csi_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.eks.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${replace(aws_iam_openid_connect_provider.eks.url, "https://", "")}:sub"
      values   = ["system:serviceaccount:kube-system:ebs-csi-controller-sa"]
    }

    condition {
      test     = "StringEquals"
      variable = "${replace(aws_iam_openid_connect_provider.eks.url, "https://", "")}:aud"
      values   = ["sts.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "ebs_csi" {
  name               = "${var.prefix}-ebs-csi-role"
  assume_role_policy = data.aws_iam_policy_document.ebs_csi_assume_role.json
}

resource "aws_iam_role_policy_attachment" "ebs_csi" {
  role       = aws_iam_role.ebs_csi.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
}

# ---------- EBS CSI driver addon ----------
#
# service_account_role_arn ties the addon's controller pod to the
# IAM role above. EKS auto-annotates the kube-system/ebs-csi-controller-sa
# ServiceAccount with this ARN, so when the controller pod starts it
# picks up STS-vended credentials with EBS permissions and stops
# being a generic "no AWS access" pod.
#
# resolve_conflicts settings: OVERWRITE on both create and update
# means terraform-managed config wins over any in-cluster drift. We
# don't expect drift here (nobody should `kubectl edit` an addon),
# but the explicit setting prevents a no-op kubectl change from
# blocking a future terraform apply.
#
# depends_on: the addon controller schedules onto the worker nodes,
# so the node group must exist before the addon can become Active.

resource "aws_eks_addon" "ebs_csi" {
  cluster_name             = aws_eks_cluster.this.name
  addon_name               = "aws-ebs-csi-driver"
  service_account_role_arn = aws_iam_role.ebs_csi.arn

  resolve_conflicts_on_create = "OVERWRITE"
  resolve_conflicts_on_update = "OVERWRITE"

  depends_on = [aws_eks_node_group.system]
}
