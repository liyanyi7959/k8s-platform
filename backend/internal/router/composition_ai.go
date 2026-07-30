package router

import (
	aigateway "k8s-platform-backend/internal/ai/adapters/gateway"
	aihttp "k8s-platform-backend/internal/ai/adapters/http"
	aimysql "k8s-platform-backend/internal/ai/adapters/mysql"
	aiapp "k8s-platform-backend/internal/ai/application"
	kopsruntime "k8s-platform-backend/internal/kops/adapters/runtime"
	kopsapp "k8s-platform-backend/internal/kops/application"
	orchestrationai "k8s-platform-backend/internal/orchestration/ai"
)

func buildAIModule(d Deps, runtime moduleRuntime, change changeModule) aiModule {
	aiRepository := aimysql.NewRepository(d.DB)
	workloadAction := kopsapp.NewActionProposalService(kopsruntime.NewActionProposalRuntime(runtime.k8s, runtime.nodeOperations, runtime.podOperations, runtime.workloadOperations, runtime.manifestApply))
	actions := orchestrationai.NewActionRuntimeWithChangeService(d.DB, workloadAction, change.application)
	resourceInspection := kopsapp.NewInspectionService(kopsruntime.NewInspectionRuntime(runtime.k8s, runtime.nodeOperations, runtime.workloadOperations, runtime.podStreams, runtime.namespaceDiagnosis, runtime.namespaceWorkloads))
	resourceQuery := aiapp.NewResourceQueryService(
		orchestrationai.NewResourceQueryRuntime(runtime.k8s, runtime.nodeOperations, runtime.podStreams),
		orchestrationai.NewResourceQueryPresenter(),
	)
	fileService := aiapp.NewAIFileService(aiRepository, d.AIUploadDir)
	toolRegistry := orchestrationai.NewToolRegistry(
		d.DB,
		aiapp.NewClusterReadModelService(orchestrationai.NewClusterReadPort(runtime.dashboard)),
		runtime.namespaceDiagnosis,
		resourceInspection,
		resourceQuery,
		actions,
		aiapp.NewResourceExportPolicyService(),
	)
	toolService := aiapp.NewToolService(aiRepository, toolRegistry)
	chatRuntime := orchestrationai.NewChatRuntime(
		d.DB,
		aigateway.NewAIGatewayService(d.DB, d.EncryptionKey),
		toolService,
		actions,
		fileService,
	)
	providerService := aiapp.NewAIProviderService(aiRepository, d.EncryptionKey)
	routeSettingsService := aiapp.NewAIRouteSettingsService(aiRepository)
	conversationService := aiapp.NewConversationService(aiRepository)
	conversationDetailService := aiapp.NewConversationDetailService(aiRepository, orchestrationai.NewConversationProjection(toolService, actions))
	runtimeAdapter := orchestrationai.NewRuntime(
		conversationDetailService,
		chatRuntime,
		toolService,
		actions,
	)
	return aiModule{runtime: aihttp.NewRuntimeController(runtimeAdapter, fileService), management: aihttp.NewManagementController(providerService, routeSettingsService, conversationService)}
}
