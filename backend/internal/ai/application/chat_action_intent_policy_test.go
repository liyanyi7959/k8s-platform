package application

import "testing"

func TestBuildAutoActionProposalSpecsRequiresExplicitScopedIntent(t *testing.T) {
	request := ChatActionIntentRequest{ResourceKind: "Deployment", Namespace: "payments", ResourceName: "api"}
	items := BuildAutoActionProposalSpecs(request, "please scale replicas to 3")
	if len(items) != 1 || items[0].ProposalType != ActionTypeScaleWorkload || items[0].Payload["replicas"] != 3 {
		t.Fatalf("scale proposal = %#v", items)
	}
	if items[0].TargetResource.Kind != "Deployment" || items[0].TargetResource.Namespace != "payments" || items[0].TargetResource.Name != "api" {
		t.Fatalf("proposal target = %#v", items[0].TargetResource)
	}

	if items := BuildAutoActionProposalSpecs(request, "should we restart this deployment?"); len(items) != 0 {
		t.Fatalf("tentative request must not create proposal: %#v", items)
	}
	if items := BuildAutoActionProposalSpecs(ChatActionIntentRequest{ResourceKind: "Deployment", ResourceName: "api"}, "restart api"); len(items) != 0 {
		t.Fatalf("namespaced workload without namespace must not create proposal: %#v", items)
	}
}

func TestBuildAutoActionProposalSpecsSupportsNodeAndImageActions(t *testing.T) {
	node := BuildAutoActionProposalSpecs(ChatActionIntentRequest{ResourceKind: "Node", ResourceName: "worker-1"}, "drain node")
	if len(node) != 1 || node[0].ProposalType != ActionTypeDrainNode || node[0].TargetResource.Namespace != "" {
		t.Fatalf("node proposal = %#v", node)
	}

	image := BuildAutoActionProposalSpecs(ChatActionIntentRequest{ResourceKind: "StatefulSet", Namespace: "data", ResourceName: "mysql"}, "update image: mysql:8.4")
	if len(image) != 1 || image[0].ProposalType != ActionTypeUpdateWorkloadImage || image[0].Payload["new_image"] != "mysql:8.4" {
		t.Fatalf("image proposal = %#v", image)
	}
}
