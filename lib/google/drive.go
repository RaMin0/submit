package google

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/ramin0/submit/config"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

var (
	_driveService *drive.Service
)

func driveService() (*drive.Service, error) {
	if _driveService == nil {
		c, err := googleClient()
		if err != nil {
			return nil, err
		}

		_driveService, err = drive.New(c)
		if err != nil {
			return nil, err
		}
	}

	return _driveService, nil
}

// DriveSubmit func
func DriveSubmit(userData map[string]string, file io.Reader, fileName string) (string, error) {
	service, err := driveService()
	if err != nil {
		return "", err
	}

	var descriptionBuffer bytes.Buffer
	template.Must(template.New("").Parse(config.SubmissionsMetaDescription)).Execute(&descriptionBuffer, userData)
	description := descriptionBuffer.String()

	folderMeta := &drive.File{
		Name:        userData["Team"],
		Description: description,
		MimeType:    "application/vnd.google-apps.folder",
		Parents:     []string{config.SubmissionsFolderID},
	}

	fileList, err := service.Files.
		List().
		Fields("files(id,webViewLink)").
		PageSize(1).
		Q(fmt.Sprintf("mimeType = '%s' and name = '%s' and '%s' in parents and trashed = false",
			folderMeta.MimeType, folderMeta.Name, folderMeta.Parents[0])).
		Do()
	if err != nil {
		return "", err
	}
	folderFound := len(fileList.Files) == 1

	if !folderFound {
		folder, err := service.Files.Create(folderMeta).Fields("id,name,webViewLink").Do()
		if err != nil {
			return "", err
		}

		if _, err = service.Permissions.Create(folder.Id, &drive.Permission{Role: "reader", Type: "anyone"}).Do(); err != nil {
			return "", err
		}

		fileList.Files = append(fileList.Files, folder)
	}

	folder := fileList.Files[0]

	uniqueFileName := time.Now().Format("20060102150405")
	fileExt := filepath.Ext(fileName)
	if fileExt != "" {
		uniqueFileName += fileExt
	}
	fileMeta := &drive.File{
		Name:        fmt.Sprintf("%s-%s", strings.Replace(userData["Team"], " ", "_", -1), uniqueFileName),
		Description: fmt.Sprintf("Original Filename:\n- %s\n\n%s", filepath.Base(fileName), description),
		Parents:     []string{folder.Id},
	}

	_, err = service.Files.Create(fileMeta).Media(file, googleapi.ChunkSize(googleapi.MinUploadChunkSize)).Do()
	if err != nil {
		return "", err
	}

	shareURL, _ := url.Parse(folder.WebViewLink)
	shareURLQuery := shareURL.Query()
	shareURLQuery.Add("hl", "en")
	shareURLQuery.Del("usp")
	shareURL.RawQuery = shareURLQuery.Encode()
	return shareURL.String(), nil
}
