package router

import (
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/middleware"
)

func registerConfigStorageRoutes(a k8sRouteArgs) {
	k8s, configuration, storage, platform, relationships, p := a.k8s, a.configuration, a.storage, a.platform, a.relationships, a.perm

	k8s.GET("/clusters/:id/configmaps", p.read, configuration.List(kopsapp.ConfigurationConfigMap))
	k8s.PATCH("/clusters/:id/configmaps/edit", p.write, configuration.EditConfigMap)
	k8s.DELETE("/clusters/:id/configmaps/:ns/:name", p.write, configuration.Delete(kopsapp.ConfigurationConfigMap))
	k8s.GET("/clusters/:id/configmaps/:ns/:name/yaml", p.read, configuration.YAML(kopsapp.ConfigurationConfigMap))
	k8s.GET("/clusters/:id/configmaps/:ns/:name/related", p.read, configuration.Related(kopsapp.ConfigurationConfigMap))

	k8s.GET("/clusters/:id/secrets", p.read, configuration.List(kopsapp.ConfigurationSecret))
	k8s.PATCH("/clusters/:id/secrets/edit", p.write, configuration.EditSecret)
	k8s.DELETE("/clusters/:id/secrets/:ns/:name", p.write, configuration.Delete(kopsapp.ConfigurationSecret))
	k8s.GET("/clusters/:id/secrets/:ns/:name/reveal", p.secretReveal, configuration.RevealSecret)
	k8s.GET("/clusters/:id/secrets/:ns/:name/decoded-data", p.secretReveal, configuration.RevealSecret)
	k8s.GET("/clusters/:id/secrets/:ns/:name/yaml", p.read, configuration.YAML(kopsapp.ConfigurationSecret))
	k8s.GET("/clusters/:id/secrets/:ns/:name/related", p.read, configuration.Related(kopsapp.ConfigurationSecret))

	k8s.GET("/clusters/:id/serviceaccounts", p.read, platform.List(kopsapp.PlatformServiceAccount))
	k8s.PATCH("/clusters/:id/serviceaccounts/edit", p.write, platform.Apply(kopsapp.PlatformServiceAccount))
	k8s.DELETE("/clusters/:id/serviceaccounts/:ns/:name", p.write, platform.Delete(kopsapp.PlatformServiceAccount))
	k8s.GET("/clusters/:id/serviceaccounts/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformServiceAccount))

	k8s.GET("/clusters/:id/pvcs", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), storage.List(kopsapp.StoragePersistentVolumeClaim))
	k8s.POST("/clusters/:id/pvcs", p.write, storage.CreatePVC)
	k8s.DELETE("/clusters/:id/pvcs/:ns/:name", p.write, storage.Delete(kopsapp.StoragePersistentVolumeClaim))
	k8s.GET("/clusters/:id/pvcs/:ns/:name/yaml", p.read, storage.YAML(kopsapp.StoragePersistentVolumeClaim))

	k8s.GET("/clusters/:id/pvs", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), storage.List(kopsapp.StoragePersistentVolume))
	k8s.DELETE("/clusters/:id/pvs/:name", p.write, storage.Delete(kopsapp.StoragePersistentVolume))
	k8s.GET("/clusters/:id/pvs/:name/yaml", p.read, storage.YAML(kopsapp.StoragePersistentVolume))

	k8s.GET("/clusters/:id/storageclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), storage.List(kopsapp.StorageClass))
	k8s.PATCH("/clusters/:id/storageclasses/edit", p.write, storage.Apply(kopsapp.StorageClass))
	k8s.DELETE("/clusters/:id/storageclasses/:name", p.write, storage.Delete(kopsapp.StorageClass))
	k8s.GET("/clusters/:id/storageclasses/:name/yaml", p.read, storage.YAML(kopsapp.StorageClass))

	k8s.GET("/clusters/:id/csidrivers", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformCSIDriver))
	k8s.PATCH("/clusters/:id/csidrivers/edit", p.write, platform.Apply(kopsapp.PlatformCSIDriver))
	k8s.DELETE("/clusters/:id/csidrivers/:name", p.write, platform.Delete(kopsapp.PlatformCSIDriver))
	k8s.GET("/clusters/:id/csidrivers/:name/yaml", p.read, platform.YAML(kopsapp.PlatformCSIDriver))

	k8s.GET("/clusters/:id/csinodes", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformCSINode))
	k8s.PATCH("/clusters/:id/csinodes/edit", p.write, platform.Apply(kopsapp.PlatformCSINode))
	k8s.DELETE("/clusters/:id/csinodes/:name", p.write, platform.Delete(kopsapp.PlatformCSINode))
	k8s.GET("/clusters/:id/csinodes/:name/yaml", p.read, platform.YAML(kopsapp.PlatformCSINode))

	k8s.GET("/clusters/:id/csistoragecapacities", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformCSIStorageCapacity))
	k8s.PATCH("/clusters/:id/csistoragecapacities/edit", p.write, platform.Apply(kopsapp.PlatformCSIStorageCapacity))
	k8s.DELETE("/clusters/:id/csistoragecapacities/:ns/:name", p.write, platform.Delete(kopsapp.PlatformCSIStorageCapacity))
	k8s.GET("/clusters/:id/csistoragecapacities/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformCSIStorageCapacity))

	k8s.GET("/clusters/:id/volumeattachments", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), relationships.List(kopsapp.RelationshipVolumeAttachment))
	k8s.PATCH("/clusters/:id/volumeattachments/edit", p.write, relationships.Apply(kopsapp.RelationshipVolumeAttachment))
	k8s.DELETE("/clusters/:id/volumeattachments/:name", p.write, relationships.Delete(kopsapp.RelationshipVolumeAttachment))
	k8s.GET("/clusters/:id/volumeattachments/:name/yaml", p.read, relationships.YAML(kopsapp.RelationshipVolumeAttachment))

	k8s.GET("/clusters/:id/volumesnapshots", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), storage.List(kopsapp.StorageVolumeSnapshot))
	k8s.PATCH("/clusters/:id/volumesnapshots/edit", p.write, storage.Apply(kopsapp.StorageVolumeSnapshot))
	k8s.DELETE("/clusters/:id/volumesnapshots/:ns/:name", p.write, storage.Delete(kopsapp.StorageVolumeSnapshot))
	k8s.GET("/clusters/:id/volumesnapshots/:ns/:name/yaml", p.read, storage.YAML(kopsapp.StorageVolumeSnapshot))

	k8s.GET("/clusters/:id/volumesnapshotclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), storage.List(kopsapp.StorageVolumeSnapshotClass))
	k8s.PATCH("/clusters/:id/volumesnapshotclasses/edit", p.write, storage.Apply(kopsapp.StorageVolumeSnapshotClass))
	k8s.DELETE("/clusters/:id/volumesnapshotclasses/:name", p.write, storage.Delete(kopsapp.StorageVolumeSnapshotClass))
	k8s.GET("/clusters/:id/volumesnapshotclasses/:name/yaml", p.read, storage.YAML(kopsapp.StorageVolumeSnapshotClass))

	k8s.GET("/clusters/:id/volumesnapshotcontents", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), storage.List(kopsapp.StorageVolumeSnapshotContent))
	k8s.PATCH("/clusters/:id/volumesnapshotcontents/edit", p.write, storage.Apply(kopsapp.StorageVolumeSnapshotContent))
	k8s.DELETE("/clusters/:id/volumesnapshotcontents/:name", p.write, storage.Delete(kopsapp.StorageVolumeSnapshotContent))
	k8s.GET("/clusters/:id/volumesnapshotcontents/:name/yaml", p.read, storage.YAML(kopsapp.StorageVolumeSnapshotContent))
}
