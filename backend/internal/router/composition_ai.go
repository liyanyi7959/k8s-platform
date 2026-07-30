package router

import (
	aigateway "k8s-platform-backend/internal/ai/adapters/gateway"
	aihttp "k8s-platform-backend/internal/ai/adapters/http"
	aimysql "k8s-platform-backend/internal/ai/adapters/mysql"
	aiapp "k8s-platform-backend/internal/ai/application"
	legacyai "k8s-platform-backend/internal/integration/ai"
	kopsruntime "k8s-platform-backend/internal/kops/adapters/runtime"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

func buildAIModule(d Deps, runtime moduleRuntime, change changeModule) aiModule {
	aiRepository := aimysql.NewRepository(d.DB)
	workloadAction := kopsapp.NewActionProposalService(kopsruntime.NewActionProposalRuntime(runtime.k8s, runtime.nodeOperations, runtime.podOperations, runtime.workloadOperations, runtime.manifestApply))
	actions := legacyai.NewActionRuntimeWithChangeService(d.DB, workloadAction, change.application)
	resourceInspection := kopsapp.NewInspectionService(kopsruntime.NewInspectionRuntime(runtime.k8s, runtime.nodeOperations, runtime.workloadOperations, runtime.podStreams, runtime.namespaceDiagnosis, runtime.namespaceWorkloads))
	resourceQuery := aiapp.NewResourceQueryService(
		legacyai.NewResourceQueryRuntime(runtime.k8s, runtime.nodeOperations, runtime.podStreams),
		legacyai.NewResourceQueryPresenter(),
	)
	fileService := aiapp.NewAIFileService(aiRepository, d.AIUploadDir)
	toolRegistry := legacyai.NewToolRegistry(
		d.DB,
		aiapp.NewClusterReadModelService(legacyai.NewClusterReadPort(runtime.dashboard)),
		runtime.namespaceDiagnosis,
		resourceInspection,
		resourceQuery,
		actions,
		aiapp.NewResourceExportPolicyService(),
	)
	toolService := aiapp.NewToolService(aiRepository, toolRegistry)
	chatRuntime := legacyai.NewChatRuntime(
		d.DB,
		aigateway.NewAIGatewayService(d.DB, d.EncryptionKey),
		toolService,
		actions,
		fileService,
	)
	providerService := aiapp.NewAIProviderService(aiRepository, d.EncryptionKey)
	routeSettingsService := aiapp.NewAIRouteSettingsService(aiRepository)
	conversationService := aiapp.NewConversationService(aiRepository)
	conversationDetailService := aiapp.NewConversationDetailService(aiRepository, legacyai.NewConversationProjection(toolService, actions))
	runtimeAdapter := legacyai.NewRuntime(
		conversationDetailService,
		chatRuntime,
		toolService,
		actions,
	)
	return aiModule{runtime: aihttp.NewRuntimeController(runtimeAdapter, fileService), management: aihttp.NewManagementController(providerService, routeSettingsService, conversationService)}
}
