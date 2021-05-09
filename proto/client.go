package proto

import (
	context "context"
	"fmt"
	"io"
	"log"
	sync "sync"
	"time"

	"github.com/algao1/imgrepo"
)

const _ChunkSize = 128 * 1024
const _PageSize = 10

type ImageRepoClient struct {
	Owner string
	Token string

	client RepoClient
	mu     sync.RWMutex
}

var _ imgrepo.ImageClient = (*ImageRepoClient)(nil)

func NewImageRepoClient(client RepoClient) *ImageRepoClient {
	return &ImageRepoClient{client: client}
}

func (irc *ImageRepoClient) Register(username, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	irc.mu.RLock()
	defer irc.mu.RUnlock()

	req := &RegisterRequest{Username: username, Password: password}

	_, err := irc.client.Register(ctx, req)
	if err != nil {
		return fmt.Errorf("%v.Register(_) = _, %v: ", irc.client, err)
	}

	return nil
}

func (irc *ImageRepoClient) Login(username, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	irc.mu.Lock()
	defer irc.mu.Unlock()

	req := &LoginRequest{Username: username, Password: password}

	resp, err := irc.client.Login(ctx, req)
	if err != nil {
		return fmt.Errorf("%v.Login(_) = _, %v: ", irc.client, err)
	}

	irc.Owner = username
	irc.Token = resp.Token

	return nil
}

func (irc *ImageRepoClient) Upload(image *imgrepo.Image) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	irc.mu.RLock()
	defer irc.mu.RUnlock()

	stream, err := irc.client.UploadImage(ctx)
	if err != nil {
		log.Fatalf("%v.UploadImage(_) = _, %v", irc.client, err)
	}

	finfo := Upload{
		Event: &Upload_Info{
			Info: &Upload_UploadInfo{
				Token: irc.Token,
				FileInfo: &FileInfo{
					FileName: image.Name,
					Owner:    image.Owner,
					Access:   int32(image.Access),
				},
			},
		},
	}

	if err := stream.Send(&finfo); err != nil {
		log.Fatalf("%v.Send(%v) = %v", stream, image.Name, err)
	}

	for cByte := 0; cByte < len(image.Raw); cByte += _ChunkSize {
		var chunk []byte
		if cByte+_ChunkSize > len(image.Raw) {
			chunk = image.Raw[cByte:]
		} else {
			chunk = image.Raw[cByte : cByte+_ChunkSize]
		}

		uchunk := Upload{
			Event: &Upload_Chunk_{
				Chunk: &Upload_Chunk{
					Chunk: chunk,
				},
			},
		}

		if err := stream.Send(&uchunk); err != nil {
			return fmt.Errorf("%v.Send(%v) = %v", stream, image.Name, err)
		}
	}

	_, err = stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("%v.CloseAndRecv() got error %v, want %v", stream, err, nil)
	}

	return nil
}

func (irc *ImageRepoClient) Download(id string) (*imgrepo.Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	irc.mu.RLock()
	defer irc.mu.RUnlock()

	req := &DownloadRequest{
		Token:  irc.Token,
		Sender: irc.Owner,
		Id:     id,
	}

	stream, err := irc.client.DownloadImage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%v.DownloadImage(_) = _, %v", irc.client, err)
	}

	img := imgrepo.Image{}

	for {
		dl, err := stream.Recv()
		if err == io.EOF {
			return &img, nil
		}
		if err != nil {
			return nil, fmt.Errorf("%v.DownloadImage(_) = _, %v", irc.client, err)
		}

		// Handles the 2 types of events (UploadInfo & Chunk).
		switch dl.GetEvent().(type) {
		case *Download_FileInfo:
			finfo := dl.GetFileInfo()

			img.Id = finfo.Id
			img.Name = finfo.FileName
			img.Owner = finfo.Owner
			img.Access = imgrepo.Permission(finfo.Access)

		case *Download_Chunk:
			img.Raw = append(img.Raw, dl.GetChunk()...)
		}
	}
}

func (irc *ImageRepoClient) List(lastId string) ([]*imgrepo.Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &ListRequest{
		Token:  irc.Token,
		Sender: irc.Owner,
		Size:   int32(_PageSize),
		LastId: lastId,
	}

	resp, err := irc.client.ListImages(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%v.ListImages(_) = _, %v: ", irc.client, err)
	}

	imgs := make([]*imgrepo.Image, len(resp.Files))
	for idx, img := range resp.Files {
		imgs[idx] = &imgrepo.Image{
			Id:     img.Id,
			Name:   img.FileName,
			Owner:  img.Owner,
			Access: imgrepo.Permission(img.Access),
		}
	}

	return imgs, nil
}
