package upload

type Website interface {
	UploadReplay(filePath string) error
}