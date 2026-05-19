package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type ServerGroupEnvItem struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type ServerGroupRegionItem struct {
	ID   uint64               `json:"id"`
	Name string               `json:"name"`
	Envs []ServerGroupEnvItem `json:"envs"`
}

type ServerGroupsResponse struct {
	Regions []ServerGroupRegionItem `json:"regions"`
}

type ServerGroupService struct {
	db *gorm.DB
}

func NewServerGroupService(db *gorm.DB) *ServerGroupService {
	return &ServerGroupService{db: db}
}

func (s *ServerGroupService) ListServerGroups(ctx context.Context) (ServerGroupsResponse, error) {
	if s.db == nil {
		return ServerGroupsResponse{}, errors.New("db is required")
	}

	var regions []model.ServerGroupRegion
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("name asc").
		Find(&regions).Error; err != nil {
		return ServerGroupsResponse{}, err
	}

	regionIDs := make([]uint64, 0, len(regions))
	for i := range regions {
		regionIDs = append(regionIDs, regions[i].ID)
	}

	envByRegion := map[uint64][]ServerGroupEnvItem{}
	if len(regionIDs) > 0 {
		var envs []model.ServerGroupEnv
		if err := s.db.WithContext(ctx).
			Where("deleted_at IS NULL AND region_id IN ?", regionIDs).
			Order("name asc").
			Find(&envs).Error; err != nil {
			return ServerGroupsResponse{}, err
		}
		for i := range envs {
			e := envs[i]
			envByRegion[e.RegionID] = append(envByRegion[e.RegionID], ServerGroupEnvItem{ID: e.ID, Name: e.Name})
		}
	}

	out := make([]ServerGroupRegionItem, 0, len(regions))
	for i := range regions {
		r := regions[i]
		envs := envByRegion[r.ID]
		if envs == nil {
			envs = []ServerGroupEnvItem{}
		}
		sort.Slice(envs, func(i, j int) bool { return envs[i].Name < envs[j].Name })
		out = append(out, ServerGroupRegionItem{ID: r.ID, Name: r.Name, Envs: envs})
	}
	return ServerGroupsResponse{Regions: out}, nil
}

func (s *ServerGroupService) CreateRegion(ctx context.Context, name string) (uint64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	n := strings.TrimSpace(name)
	if n == "" {
		return 0, ErrInvalidParams
	}

	var created model.ServerGroupRegion
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.ServerGroupRegion
		if err := tx.Select("id").
			Where("deleted_at IS NULL AND LOWER(name) = LOWER(?)", n).
			First(&existing).Error; err == nil {
			return ErrConflict
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		created = model.ServerGroupRegion{Name: n}
		return tx.Create(&created).Error
	})
	if err != nil {
		return 0, err
	}
	return created.ID, nil
}

func (s *ServerGroupService) PatchRegion(ctx context.Context, id uint64, name string) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrInvalidParams
	}
	n := strings.TrimSpace(name)
	if n == "" {
		return ErrInvalidParams
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var region model.ServerGroupRegion
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&region).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		oldName := region.Name

		var existing model.ServerGroupRegion
		if err := tx.Select("id").
			Where("deleted_at IS NULL AND id <> ? AND LOWER(name) = LOWER(?)", id, n).
			First(&existing).Error; err == nil {
			return ErrConflict
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if strings.EqualFold(strings.TrimSpace(oldName), n) {
			return tx.Model(&model.ServerGroupRegion{}).Where("id = ? AND deleted_at IS NULL", id).Update("name", n).Error
		}

		if err := tx.Model(&model.ServerGroupRegion{}).Where("id = ? AND deleted_at IS NULL", id).Update("name", n).Error; err != nil {
			return err
		}
		return updateServerTagsForRegionRename(ctx, tx, oldName, n)
	})
}

func (s *ServerGroupService) DeleteRegion(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrInvalidParams
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var region model.ServerGroupRegion
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&region).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		var envs []model.ServerGroupEnv
		if err := tx.Where("deleted_at IS NULL AND region_id = ?", id).Find(&envs).Error; err != nil {
			return err
		}
		removedEnvNames := map[string]bool{}
		for i := range envs {
			removedEnvNames[strings.ToLower(strings.TrimSpace(envs[i].Name))] = true
		}

		now := time.Now().UTC()
		if err := tx.Model(&model.ServerGroupEnv{}).
			Where("region_id = ? AND deleted_at IS NULL", id).
			Update("deleted_at", &now).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.ServerGroupRegion{}).
			Where("id = ? AND deleted_at IS NULL", id).
			Update("deleted_at", &now).Error; err != nil {
			return err
		}
		return updateServerTagsForRegionDelete(ctx, tx, region.Name, removedEnvNames)
	})
}

func (s *ServerGroupService) CreateEnv(ctx context.Context, regionID uint64, name string) (uint64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	if regionID == 0 {
		return 0, ErrInvalidParams
	}
	n := strings.TrimSpace(name)
	if n == "" {
		return 0, ErrInvalidParams
	}

	var created model.ServerGroupEnv
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var region model.ServerGroupRegion
		if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", regionID).First(&region).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		var existing model.ServerGroupEnv
		if err := tx.Select("id").
			Where("deleted_at IS NULL AND region_id = ? AND LOWER(name) = LOWER(?)", regionID, n).
			First(&existing).Error; err == nil {
			return ErrConflict
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		created = model.ServerGroupEnv{RegionID: regionID, Name: n}
		return tx.Create(&created).Error
	})
	if err != nil {
		return 0, err
	}
	return created.ID, nil
}

func (s *ServerGroupService) PatchEnv(ctx context.Context, id uint64, name string) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrInvalidParams
	}
	n := strings.TrimSpace(name)
	if n == "" {
		return ErrInvalidParams
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var env model.ServerGroupEnv
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&env).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		oldName := env.Name

		var region model.ServerGroupRegion
		if err := tx.Where("deleted_at IS NULL AND id = ?", env.RegionID).First(&region).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		var existing model.ServerGroupEnv
		if err := tx.Select("id").
			Where("deleted_at IS NULL AND id <> ? AND region_id = ? AND LOWER(name) = LOWER(?)", id, env.RegionID, n).
			First(&existing).Error; err == nil {
			return ErrConflict
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Model(&model.ServerGroupEnv{}).Where("id = ? AND deleted_at IS NULL", id).Update("name", n).Error; err != nil {
			return err
		}
		if strings.EqualFold(strings.TrimSpace(oldName), n) {
			return nil
		}
		return updateServerTagsForEnvRename(ctx, tx, region.Name, oldName, n)
	})
}

func (s *ServerGroupService) DeleteEnv(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrInvalidParams
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var env model.ServerGroupEnv
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&env).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		var region model.ServerGroupRegion
		if err := tx.Where("deleted_at IS NULL AND id = ?", env.RegionID).First(&region).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		now := time.Now().UTC()
		if err := tx.Model(&model.ServerGroupEnv{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", &now).Error; err != nil {
			return err
		}
		return updateServerTagsForEnvDelete(ctx, tx, region.Name, env.Name)
	})
}

type serverTagRow struct {
	ID   uint64  `gorm:"column:id"`
	Tags *string `gorm:"column:tags"`
}

func updateServerTagsForRegionRename(ctx context.Context, tx *gorm.DB, oldName, newName string) error {
	oldN := strings.TrimSpace(oldName)
	newN := strings.TrimSpace(newName)
	if oldN == "" || newN == "" {
		return nil
	}
	pattern := "%\"region:" + strings.ToLower(oldN) + "\"%"
	var rows []serverTagRow
	if err := tx.WithContext(ctx).
		Model(&model.Server{}).
		Select("id", "tags").
		Where("deleted_at IS NULL AND LOWER(tags) LIKE ?", pattern).
		Find(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		tags := parseTagsJSON(rows[i].Tags)
		v, ok := readPrefixedTag(tags, "region:")
		if !ok || !strings.EqualFold(v, oldN) {
			continue
		}
		next := setPrefixedTag(tags, "region:", newN)
		encoded := mustMarshalTags(next)
		if err := tx.WithContext(ctx).Model(&model.Server{}).Where("id = ?", rows[i].ID).Update("tags", encoded).Error; err != nil {
			return err
		}
	}
	return nil
}

func updateServerTagsForEnvRename(ctx context.Context, tx *gorm.DB, regionName, oldEnv, newEnv string) error {
	rn := strings.TrimSpace(regionName)
	oe := strings.TrimSpace(oldEnv)
	ne := strings.TrimSpace(newEnv)
	if rn == "" || oe == "" || ne == "" {
		return nil
	}
	patternRegion := "%\"region:" + strings.ToLower(rn) + "\"%"
	patternEnv := "%\"env:" + strings.ToLower(oe) + "\"%"
	var rows []serverTagRow
	if err := tx.WithContext(ctx).
		Model(&model.Server{}).
		Select("id", "tags").
		Where("deleted_at IS NULL AND LOWER(tags) LIKE ? AND LOWER(tags) LIKE ?", patternRegion, patternEnv).
		Find(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		tags := parseTagsJSON(rows[i].Tags)
		curRegion, ok1 := readPrefixedTag(tags, "region:")
		curEnv, ok2 := readPrefixedTag(tags, "env:")
		if !ok1 || !ok2 {
			continue
		}
		if !strings.EqualFold(curRegion, rn) || !strings.EqualFold(curEnv, oe) {
			continue
		}
		next := setPrefixedTag(tags, "env:", ne)
		encoded := mustMarshalTags(next)
		if err := tx.WithContext(ctx).Model(&model.Server{}).Where("id = ?", rows[i].ID).Update("tags", encoded).Error; err != nil {
			return err
		}
	}
	return nil
}

func updateServerTagsForEnvDelete(ctx context.Context, tx *gorm.DB, regionName, envName string) error {
	rn := strings.TrimSpace(regionName)
	en := strings.TrimSpace(envName)
	if rn == "" || en == "" {
		return nil
	}
	patternRegion := "%\"region:" + strings.ToLower(rn) + "\"%"
	patternEnv := "%\"env:" + strings.ToLower(en) + "\"%"
	var rows []serverTagRow
	if err := tx.WithContext(ctx).
		Model(&model.Server{}).
		Select("id", "tags").
		Where("deleted_at IS NULL AND LOWER(tags) LIKE ? AND LOWER(tags) LIKE ?", patternRegion, patternEnv).
		Find(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		tags := parseTagsJSON(rows[i].Tags)
		curRegion, ok1 := readPrefixedTag(tags, "region:")
		curEnv, ok2 := readPrefixedTag(tags, "env:")
		if !ok1 || !ok2 {
			continue
		}
		if !strings.EqualFold(curRegion, rn) || !strings.EqualFold(curEnv, en) {
			continue
		}
		next := setPrefixedTag(tags, "env:", "")
		encoded := mustMarshalTags(next)
		if err := tx.WithContext(ctx).Model(&model.Server{}).Where("id = ?", rows[i].ID).Update("tags", encoded).Error; err != nil {
			return err
		}
	}
	return nil
}

func updateServerTagsForRegionDelete(ctx context.Context, tx *gorm.DB, regionName string, removedEnvNames map[string]bool) error {
	rn := strings.TrimSpace(regionName)
	if rn == "" {
		return nil
	}
	patternRegion := "%\"region:" + strings.ToLower(rn) + "\"%"
	var rows []serverTagRow
	if err := tx.WithContext(ctx).
		Model(&model.Server{}).
		Select("id", "tags").
		Where("deleted_at IS NULL AND LOWER(tags) LIKE ?", patternRegion).
		Find(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		tags := parseTagsJSON(rows[i].Tags)
		curRegion, ok := readPrefixedTag(tags, "region:")
		if !ok || !strings.EqualFold(curRegion, rn) {
			continue
		}
		next := setPrefixedTag(tags, "region:", "")
		if curEnv, ok2 := readPrefixedTag(next, "env:"); ok2 {
			if removedEnvNames[strings.ToLower(strings.TrimSpace(curEnv))] {
				next = setPrefixedTag(next, "env:", "")
			}
		}
		encoded := mustMarshalTags(next)
		if err := tx.WithContext(ctx).Model(&model.Server{}).Where("id = ?", rows[i].ID).Update("tags", encoded).Error; err != nil {
			return err
		}
	}
	return nil
}

func parseTagsJSON(tagsJSON *string) []string {
	if tagsJSON == nil || strings.TrimSpace(*tagsJSON) == "" {
		return []string{}
	}
	var out []string
	if json.Unmarshal([]byte(*tagsJSON), &out) != nil {
		return []string{}
	}
	clean := make([]string, 0, len(out))
	seen := map[string]bool{}
	for _, t := range out {
		v := strings.TrimSpace(t)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		clean = append(clean, v)
	}
	return clean
}

func mustMarshalTags(tags []string) *string {
	clean := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, t := range tags {
		v := strings.TrimSpace(t)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		clean = append(clean, v)
	}
	b, _ := json.Marshal(clean)
	s := string(b)
	return &s
}

func readPrefixedTag(tags []string, prefix string) (string, bool) {
	for _, t := range tags {
		if strings.HasPrefix(t, prefix) {
			v := strings.TrimSpace(strings.TrimPrefix(t, prefix))
			if v == "" {
				return "", false
			}
			return v, true
		}
	}
	return "", false
}

func setPrefixedTag(tags []string, prefix string, value string) []string {
	out := make([]string, 0, len(tags)+1)
	seen := map[string]bool{}
	for _, t := range tags {
		tv := strings.TrimSpace(t)
		if tv == "" || strings.HasPrefix(tv, prefix) {
			continue
		}
		if !seen[tv] {
			seen[tv] = true
			out = append(out, tv)
		}
	}
	v := strings.TrimSpace(value)
	if v != "" {
		joined := prefix + v
		if !seen[joined] {
			out = append(out, joined)
		}
	}
	return out
}
