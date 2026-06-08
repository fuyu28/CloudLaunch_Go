package services

import (
	"sort"
	"strings"
	"time"

	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/infrastructure/storage"
)

func composeCloudGameMetadata(game domain.Game) storage.CloudGameMetadata {
	return storage.CloudGameMetadata{
		ID:             game.ID,
		Title:          game.Title,
		Publisher:      game.Publisher,
		PlayStatus:     string(game.PlayStatus),
		TotalPlayTime:  game.TotalPlayTime,
		LastPlayed:     game.LastPlayed,
		ClearedAt:      game.ClearedAt,
		CurrentRouteID: game.CurrentRouteID,
		CreatedAt:      game.CreatedAt,
		UpdatedAt:      game.UpdatedAt,
	}
}

func composeCloudSessions(sessions []domain.PlaySession) []storage.CloudSessionRecord {
	records := make([]storage.CloudSessionRecord, 0, len(sessions))
	for _, session := range sessions {
		records = append(records, storage.CloudSessionRecord{
			ID:          session.ID,
			PlayedAt:    session.PlayedAt,
			Duration:    session.Duration,
			SessionName: session.SessionName,
			UpdatedAt:   session.UpdatedAt,
		})
	}
	return records
}

func composeLocalPlaySession(gameID string, session storage.CloudSessionRecord) domain.PlaySession {
	return domain.PlaySession{
		ID:          session.ID,
		GameID:      gameID,
		PlayedAt:    session.PlayedAt,
		Duration:    session.Duration,
		SessionName: session.SessionName,
		UpdatedAt:   session.UpdatedAt,
	}
}

func composeSyncedLocalGame(
	cloud storage.CloudGameMetadata,
	local *domain.Game,
	imagePath *string,
) domain.Game {
	exePath := UnconfiguredExePath
	saveFolder := (*string)(nil)
	localSaveHash := (*string)(nil)
	localSaveHashUpdatedAt := (*time.Time)(nil)
	if local != nil {
		if strings.TrimSpace(local.ExePath) != "" {
			exePath = local.ExePath
		}
		saveFolder = local.SaveFolderPath
		localSaveHash = local.LocalSaveHash
		localSaveHashUpdatedAt = local.LocalSaveHashUpdatedAt
	}

	return domain.Game{
		ID:                     cloud.ID,
		Title:                  cloud.Title,
		Publisher:              cloud.Publisher,
		ImagePath:              imagePath,
		ExePath:                exePath,
		SaveFolderPath:         saveFolder,
		CreatedAt:              cloud.CreatedAt,
		UpdatedAt:              cloud.UpdatedAt,
		LocalSaveHash:          localSaveHash,
		LocalSaveHashUpdatedAt: localSaveHashUpdatedAt,
		PlayStatus:             domain.PlayStatus(cloud.PlayStatus),
		TotalPlayTime:          cloud.TotalPlayTime,
		LastPlayed:             cloud.LastPlayed,
		ClearedAt:              cloud.ClearedAt,
		CurrentRouteID:         cloud.CurrentRouteID,
	}
}

func mergeSessions(localSessions []domain.PlaySession, cloudSessions []storage.CloudSessionRecord) mergedSessionsResult {
	merged := make(map[string]storage.CloudSessionRecord, len(localSessions)+len(cloudSessions))
	localMap := make(map[string]domain.PlaySession, len(localSessions))
	cloudMap := make(map[string]storage.CloudSessionRecord, len(cloudSessions))
	uploadedCount := 0
	downloadedCount := 0
	changed := false

	for _, session := range localSessions {
		localMap[session.ID] = session
		merged[session.ID] = storage.CloudSessionRecord{
			ID:          session.ID,
			PlayedAt:    session.PlayedAt,
			Duration:    session.Duration,
			SessionName: session.SessionName,
			UpdatedAt:   session.UpdatedAt,
		}
	}
	for _, session := range cloudSessions {
		cloudMap[session.ID] = session
		existing, ok := localMap[session.ID]
		if !ok {
			merged[session.ID] = session
			downloadedCount++
			changed = true
			continue
		}
		if session.UpdatedAt.After(existing.UpdatedAt) {
			if !sessionsEquivalent(existing, session) {
				merged[session.ID] = session
				downloadedCount++
				changed = true
			}
			continue
		}
		if existing.UpdatedAt.After(session.UpdatedAt) {
			if !sessionsEquivalent(existing, session) {
				uploadedCount++
				changed = true
			}
			continue
		}
		if !sessionsEquivalent(existing, session) {
			uploadedCount++
			changed = true
		}
	}
	for _, session := range localSessions {
		if _, ok := cloudMap[session.ID]; ok {
			continue
		}
		uploadedCount++
		changed = true
	}

	result := make([]storage.CloudSessionRecord, 0, len(merged))
	for _, session := range merged {
		result = append(result, session)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].PlayedAt.Equal(result[j].PlayedAt) {
			return result[i].ID > result[j].ID
		}
		return result[i].PlayedAt.After(result[j].PlayedAt)
	})

	return mergedSessionsResult{
		Sessions:        result,
		UploadedCount:   uploadedCount,
		DownloadedCount: downloadedCount,
		Changed:         changed,
	}
}

func sessionsEquivalent(local domain.PlaySession, cloud storage.CloudSessionRecord) bool {
	return local.ID == cloud.ID &&
		local.PlayedAt.Equal(cloud.PlayedAt) &&
		local.Duration == cloud.Duration &&
		strings.TrimSpace(stringValue(local.SessionName)) == strings.TrimSpace(stringValue(cloud.SessionName)) &&
		local.UpdatedAt.Equal(cloud.UpdatedAt)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (service *CloudSyncService) mergeCloudGameMetadata(
	cloud storage.CloudGameMetadata,
	local *domain.Game,
	sessions []storage.CloudSessionRecord,
) domain.Game {
	base := domain.Game{
		ID:             cloud.ID,
		Title:          cloud.Title,
		Publisher:      cloud.Publisher,
		PlayStatus:     domain.PlayStatus(cloud.PlayStatus),
		CreatedAt:      cloud.CreatedAt,
		UpdatedAt:      cloud.UpdatedAt,
		LastPlayed:     cloud.LastPlayed,
		ClearedAt:      cloud.ClearedAt,
		CurrentRouteID: cloud.CurrentRouteID,
	}
	if local != nil && local.UpdatedAt.After(cloud.UpdatedAt) {
		base = *local
	}

	var total int64
	var lastPlayed *time.Time
	for _, session := range sessions {
		total += session.Duration
		if lastPlayed == nil || session.PlayedAt.After(*lastPlayed) {
			playedAt := session.PlayedAt
			lastPlayed = &playedAt
		}
	}
	base.TotalPlayTime = total
	base.LastPlayed = lastPlayed
	return base
}

func cloudMetadataFromGame(game domain.Game, imageKey *string) storage.CloudGameMetadata {
	return storage.CloudGameMetadata{
		ID:             game.ID,
		Title:          game.Title,
		Publisher:      game.Publisher,
		ImageKey:       imageKey,
		PlayStatus:     string(game.PlayStatus),
		TotalPlayTime:  game.TotalPlayTime,
		LastPlayed:     game.LastPlayed,
		ClearedAt:      game.ClearedAt,
		CurrentRouteID: game.CurrentRouteID,
		CreatedAt:      game.CreatedAt,
		UpdatedAt:      game.UpdatedAt,
	}
}
