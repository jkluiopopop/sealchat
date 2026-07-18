package service

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sealchat/model"
	"sealchat/utils"
)

func TestExtractTheaterPackageZIPRejectsTraversalAndSymlink(t *testing.T) {
	tests := []struct {
		name string
		path string
		mode os.FileMode
	}{
		{name: "traversal", path: "../escape.txt"},
		{name: "absolute", path: "/escape.txt"},
		{name: "symlink", path: "link", mode: os.ModeSymlink | 0o777},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archivePath := filepath.Join(t.TempDir(), "package.zip")
			file, err := os.Create(archivePath)
			if err != nil {
				t.Fatal(err)
			}
			writer := zip.NewWriter(file)
			header := &zip.FileHeader{Name: test.path, Method: zip.Store}
			if test.mode != 0 {
				header.SetMode(test.mode)
			}
			entry, err := writer.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := entry.Write([]byte("data")); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			if err := extractTheaterPackageZIP(archivePath, t.TempDir()); err == nil {
				t.Fatal("expected unsafe ZIP to be rejected")
			}
		})
	}
}

func TestLoadAndValidateTheaterPackageChecksHash(t *testing.T) {
	root := t.TempDir()
	documentPath := filepath.Join(root, "stage", "document.json")
	if err := os.MkdirAll(filepath.Dir(documentPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(documentPath, []byte(`{"activeSceneId":null,"liveState":{},"scenes":{},"persistentObjects":{},"characters":{},"resources":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := theaterPackageFile(documentPath, "application/json", "document.json")
	if err != nil {
		t.Fatal(err)
	}
	document.Path = "stage/document.json"
	manifest := TheaterPackageManifest{PackageVersion: theaterPackageVersion, SchemaVersion: 1, PackageID: "package-test", Document: document, Resources: []TheaterPackageResource{}, Audio: []TheaterPackageAudio{}}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAndValidateTheaterPackage(root); err != nil {
		t.Fatalf("valid package rejected: %v", err)
	}
	if err := os.WriteFile(documentPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAndValidateTheaterPackage(root); err == nil {
		t.Fatal("expected hash mismatch")
	}
}

func TestRemapTheaterPackageSnapshotCreatesIndependentReferences(t *testing.T) {
	owner := "user-old"
	identity := "identity-old"
	parent := "object-parent"
	active := "scene-old"
	snapshot := TheaterSharedSnapshot{
		ActiveSceneID: &active,
		LiveState:     json.RawMessage(`{"worldId":"world-old","channelId":"channel-old","assetId":"audio-old"}`),
		Scenes: map[string]TheaterSceneSnapshot{
			"scene-old": {
				ID: "scene-old", Name: "Scene", State: json.RawMessage(`{"resourceId":"resource-old"}`),
				Objects: map[string]TheaterObjectSnapshot{
					"object-parent": {ID: "object-parent", Kind: "group", Width: 10, Height: 10, Visible: true, Content: json.RawMessage(`{}`), Actions: json.RawMessage(`[]`), Metadata: json.RawMessage(`{}`)},
					"object-child": {
						ID: "object-child", ParentID: &parent, Kind: "image", Width: 10, Height: 10, Visible: true,
						OwnerUserID: &owner, CharacterIdentityID: &identity,
						Content: json.RawMessage(`{"resourceId":"resource-old","url":"/theater/resources/resource-old/content"}`),
						Actions: json.RawMessage(`[{"sceneId":"scene-old","objectId":"object-parent","assetId":"audio-old","resourceAttachmentId":"attachment-old","identityId":"identity-old"}]`), Metadata: json.RawMessage(`{}`),
					},
				},
			},
		},
		PersistentObjects: map[string]TheaterObjectSnapshot{}, Characters: map[string]TheaterObjectSnapshot{}, Resources: map[string]TheaterResourcePublic{},
	}
	remap := theaterPackageRemap{
		scenes:    map[string]string{"scene-old": "scene-new"},
		objects:   map[string]string{"object-parent": "parent-new", "object-child": "child-new"},
		resources: map[string]string{"resource-old": "resource-new"},
		audio:     map[string]string{"audio-old": "audio-new"}, appearance: map[string]string{}, attachments: map[string]string{"attachment-old": "attachment-new"}, worldID: "world-new", channelID: "channel-new",
	}
	result, warnings, err := remapTheaterPackageSnapshot(snapshot, remap)
	if err != nil {
		t.Fatal(err)
	}
	if result.ActiveSceneID == nil || *result.ActiveSceneID != "scene-new" {
		t.Fatalf("active scene not remapped: %#v", result.ActiveSceneID)
	}
	child := result.Scenes["scene-new"].Objects["child-new"]
	if child.ParentID == nil || *child.ParentID != "parent-new" {
		t.Fatalf("parent not remapped: %#v", child.ParentID)
	}
	if child.OwnerUserID != nil || child.CharacterIdentityID != nil {
		t.Fatal("identity ownership should be cleared")
	}
	combined := string(child.Content) + string(child.Actions) + string(result.LiveState)
	for _, expected := range []string{"resource-new", "audio-new", "attachment-new", "scene-new", "parent-new", "world-new", "channel-new"} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("missing remapped reference %q in %s", expected, combined)
		}
	}
	if len(warnings) == 0 {
		t.Fatal("expected identity remap warning")
	}
}

func TestTheaterPackageImportAppendsAndIsJobIdempotent(t *testing.T) {
	actorID, sourceWorldID, sourceChannelID := initTheaterServiceTest(t)
	storageDir := t.TempDir()
	theaterPackageWorkerState.Lock()
	theaterPackageWorkerState.config.StorageDir = storageDir
	theaterPackageWorkerState.Unlock()
	if _, err := InitStorageManager(utils.StorageConfig{Mode: utils.StorageModeLocal, Local: utils.LocalStorageConfig{UploadDir: t.TempDir(), TempDir: t.TempDir()}}); err != nil {
		t.Fatal(err)
	}

	sourceRoom, err := model.TheaterRoomCreateIfMissing(sourceWorldID, "", actorID)
	if err != nil {
		t.Fatal(err)
	}
	sourceSceneID := "source-scene-" + utils.NewIDWithLength(6)
	if err := model.GetDB().Create(&model.TheaterSceneModel{
		StringPKBaseModel: model.StringPKBaseModel{ID: sourceSceneID}, RoomID: sourceRoom.ID,
		Name: "Imported Scene", SortOrder: 1, StateJSON: `{}`, SchemaVersion: model.TheaterSchemaVersion,
		CreatedBy: actorID, UpdatedBy: actorID,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.GetDB().Create(&model.TheaterObjectModel{
		StringPKBaseModel: model.StringPKBaseModel{ID: "source-object-" + utils.NewIDWithLength(6)},
		RoomID:            sourceRoom.ID, SceneID: sourceSceneID, Kind: "group", Name: "Imported Object",
		Width: 100, Height: 100, Scale: 1, ScaleX: 1, ScaleY: 1, Visible: true,
		AspectRatioLocked: true, OrderKey: "a", ContentJSON: `{}`, ActionsJSON: `[]`, MetadataJSON: `{}`,
		SchemaVersion: model.TheaterSchemaVersion, CreatedBy: actorID, UpdatedBy: actorID,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.GetDB().Model(&model.TheaterRoomModel{}).Where("id = ?", sourceRoom.ID).Updates(map[string]any{"active_scene_id": sourceSceneID, "state_json": `{}`}).Error; err != nil {
		t.Fatal(err)
	}
	resourceBytes := []byte("theater-package-resource")
	resourceHash := sha256.Sum256(resourceBytes)
	resourceTemp := filepath.Join(t.TempDir(), "resource.bin")
	if err := os.WriteFile(resourceTemp, resourceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	location, err := PersistAttachmentFile(resourceHash[:], int64(len(resourceBytes)), resourceTemp, "application/octet-stream")
	if err != nil {
		t.Fatal(err)
	}
	attachment := model.AttachmentModel{
		Hash: model.ByteArray(resourceHash[:]), Filename: "resource.bin", Size: int64(len(resourceBytes)),
		MimeType: "application/octet-stream", UserID: actorID, ChannelID: sourceChannelID,
		StorageType: location.StorageType, ObjectKey: location.ObjectKey, ExternalURL: location.ExternalURL,
	}
	if err := model.GetDB().Create(&attachment).Error; err != nil {
		t.Fatal(err)
	}
	readyAt := time.Now()
	if err := model.GetDB().Create(&model.TheaterResourceModel{
		StringPKBaseModel: model.StringPKBaseModel{ID: "source-resource-" + utils.NewIDWithLength(6)},
		RoomID:            sourceRoom.ID, AttachmentID: attachment.ID, Kind: "file", ContentHash: hex.EncodeToString(resourceHash[:]),
		SizeBytes: int64(len(resourceBytes)), MimeType: "application/octet-stream", OriginalFilename: "resource.bin",
		Status: "ready", ProcessingProgress: 1, VariantsJSON: `[]`, CreatedBy: actorID, ReadyAt: &readyAt,
	}).Error; err != nil {
		t.Fatal(err)
	}

	exportJob := &model.TheaterPackageJobModel{
		StringPKBaseModel: model.StringPKBaseModel{ID: utils.NewID()}, Type: model.TheaterPackageJobTypeExport,
		Status: model.TheaterPackageJobStatusRunning, ActorUserID: actorID, SourceWorldID: sourceWorldID, InputChannelID: sourceChannelID,
	}
	if err := model.GetDB().Create(exportJob).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := exportTheaterPackage(t.Context(), exportJob); err != nil {
		t.Fatal(err)
	}
	if err := model.GetDB().Where("id = ?", exportJob.ID).First(exportJob).Error; err != nil {
		t.Fatal(err)
	}
	if exportJob.OutputFilePath == "" {
		t.Fatal("export path missing")
	}

	targetWorldID := "target-world-" + utils.NewIDWithLength(6)
	targetChannelID := "target-channel-" + utils.NewIDWithLength(6)
	if err := model.GetDB().Create(&model.WorldModel{
		StringPKBaseModel: model.StringPKBaseModel{ID: targetWorldID}, Name: "Target", OwnerID: actorID,
		InviteSlug: utils.NewIDWithLength(12), Status: "active",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.GetDB().Create(&model.WorldMemberModel{
		StringPKBaseModel: model.StringPKBaseModel{ID: utils.NewID()}, WorldID: targetWorldID,
		UserID: actorID, Role: model.WorldRoleOwner, JoinedAt: time.Now(),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.GetDB().Create(&model.ChannelModel{
		StringPKBaseModel: model.StringPKBaseModel{ID: targetChannelID}, WorldID: targetWorldID,
		Name: "Target Stage", Status: model.ChannelStatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}
	targetRoom, err := model.TheaterRoomCreateIfMissing(targetWorldID, "", actorID)
	if err != nil {
		t.Fatal(err)
	}
	existingSceneID := "existing-scene-" + utils.NewIDWithLength(6)
	if err := model.GetDB().Create(&model.TheaterSceneModel{
		StringPKBaseModel: model.StringPKBaseModel{ID: existingSceneID}, RoomID: targetRoom.ID,
		Name: "Existing", SortOrder: 1, StateJSON: `{}`, SchemaVersion: model.TheaterSchemaVersion,
		CreatedBy: actorID, UpdatedBy: actorID,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.GetDB().Model(&model.TheaterRoomModel{}).Where("id = ?", targetRoom.ID).Update("active_scene_id", existingSceneID).Error; err != nil {
		t.Fatal(err)
	}

	importJob := &model.TheaterPackageJobModel{
		StringPKBaseModel: model.StringPKBaseModel{ID: utils.NewID()}, Type: model.TheaterPackageJobTypeImport,
		Status: model.TheaterPackageJobStatusRunning, ActorUserID: actorID, TargetWorldID: targetWorldID,
		InputChannelID: targetChannelID, InputFilePath: exportJob.OutputFilePath,
	}
	if err := model.GetDB().Create(importJob).Error; err != nil {
		t.Fatal(err)
	}
	summary, err := importTheaterPackage(t.Context(), importJob)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Scenes != 1 || len(summary.ImportedSceneIDs) != 1 || summary.ImportedSceneIDs[0] == sourceSceneID {
		t.Fatalf("unexpected import summary: %#v", summary)
	}
	assertTheaterPackageTarget(t, targetRoom.ID, existingSceneID, 2, 1, 1)
	if _, err := importTheaterPackage(t.Context(), importJob); err != nil {
		t.Fatal(err)
	}
	assertTheaterPackageTarget(t, targetRoom.ID, existingSceneID, 2, 1, 1)

	secondJob := *importJob
	secondJob.ID = utils.NewID()
	secondJob.CreatedAt = time.Time{}
	secondJob.UpdatedAt = time.Time{}
	if err := model.GetDB().Create(&secondJob).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := importTheaterPackage(t.Context(), &secondJob); err != nil {
		t.Fatal(err)
	}
	assertTheaterPackageTarget(t, targetRoom.ID, existingSceneID, 3, 2, 2)

	var importedResource model.TheaterResourceModel
	if err := model.GetDB().Where("room_id = ?", targetRoom.ID).Order("created_at ASC").First(&importedResource).Error; err != nil {
		t.Fatal(err)
	}
	var importedAttachment model.AttachmentModel
	if err := model.GetDB().Where("id = ?", importedResource.AttachmentID).First(&importedAttachment).Error; err != nil {
		t.Fatal(err)
	}
	materialized, err := MaterializeAttachmentToTempFile(&importedAttachment)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(materialized)
	actualResourceBytes, err := os.ReadFile(materialized)
	if err != nil {
		t.Fatal(err)
	}
	if string(actualResourceBytes) != string(resourceBytes) {
		t.Fatalf("resource content mismatch: %q", actualResourceBytes)
	}
}

func assertTheaterPackageTarget(t *testing.T, roomID, activeSceneID string, scenes, objects, resources int64) {
	t.Helper()
	var room model.TheaterRoomModel
	if err := model.GetDB().Where("id = ?", roomID).First(&room).Error; err != nil {
		t.Fatal(err)
	}
	if room.ActiveSceneID != activeSceneID {
		t.Fatalf("active scene changed: got %s want %s", room.ActiveSceneID, activeSceneID)
	}
	var sceneCount, objectCount, resourceCount int64
	if err := model.GetDB().Model(&model.TheaterSceneModel{}).Where("room_id = ?", roomID).Count(&sceneCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.GetDB().Model(&model.TheaterObjectModel{}).Where("room_id = ?", roomID).Count(&objectCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := model.GetDB().Model(&model.TheaterResourceModel{}).Where("room_id = ?", roomID).Count(&resourceCount).Error; err != nil {
		t.Fatal(err)
	}
	if sceneCount != scenes || objectCount != objects || resourceCount != resources {
		t.Fatalf("unexpected target counts: scenes=%d objects=%d resources=%d", sceneCount, objectCount, resourceCount)
	}
}
