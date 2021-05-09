package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/algao1/imgrepo"
	"github.com/algao1/imgrepo/digitalocean"
	"github.com/algao1/imgrepo/mongo"
	"github.com/algao1/imgrepo/redis"

	pb "github.com/algao1/imgrepo/proto"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	serverAddr = flag.String("server_addr", "localhost:10000", "The server address in the format of host:port")
)

// _ChunkSize determines the size of each chunk.
const _ChunkSize = 128 * 1024

type repoServer struct {
	pb.UnimplementedRepoServer

	us imgrepo.UserService
	ss imgrepo.SessionService
	ir imgrepo.ImageRegistry
}

// Register registers a user account.
func (s *repoServer) Register(ctx context.Context, req *pb.RegisterRequest) (*emptypb.Empty, error) {
	return new(emptypb.Empty), s.us.Register(req.Username, req.Password)
}

// Login logs in a user account.
func (s *repoServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	err := s.us.Login(req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	// Generates a new session.
	uuid, err := s.ss.NewSession()
	if err != nil {
		return nil, err
	}

	return &pb.LoginResponse{Token: uuid}, nil
}

// UploadImage uploads an image to the image repository.
//
// It gets a stream of events (fileinfo & chunks), and responds with either
// a completion message, or an error.
func (s *repoServer) UploadImage(stream pb.Repo_UploadImageServer) error {
	img := imgrepo.Image{}
	startTime := time.Now()

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			endTime := time.Now()
			log.Printf("finished receiving file in: %.4fs\n", endTime.Sub(startTime).Seconds())

			uerr := s.ir.Upload(&img)
			if uerr != nil {
				return err
			}

			return stream.SendAndClose(new(emptypb.Empty))
		}
		if err != nil {
			return err
		}

		// Handles the 2 types of events (UploadInfo & Chunk).
		switch in.GetEvent().(type) {
		case *pb.Upload_Info:
			// Verify that the user is logged in using token.
			err := s.ss.IsSession(in.GetInfo().Token)
			if err != nil {
				return fmt.Errorf("%q: %w", "unable to authenticate UploadImage()", err)
			}

			finfo := in.GetInfo().GetFileInfo()

			img.Name = finfo.FileName
			img.Owner = finfo.Owner
			img.Access = imgrepo.Permission(finfo.Access)

			log.Println("received file info")
		case *pb.Upload_Chunk_:
			img.Raw = append(img.Raw, in.GetChunk().Chunk...)
		}
	}
}

// DownloadImage downloads an image with id specified by the request.
//
// The id is first looked up in the image registry, then downloaded from the image
// storage. Once obtained, the file is streamed back to the client.
func (s *repoServer) DownloadImage(req *pb.DownloadRequest, stream pb.Repo_DownloadImageServer) error {
	// Verify that the user is logged in using token.
	err := s.ss.IsSession(req.Token)
	if err != nil {
		return fmt.Errorf("%q: %w", "unable to authenticate DownloadImage()", err)
	}

	image, err := s.ir.Download(req.Sender, req.Id)
	if err != nil {
		return fmt.Errorf("%q: %w", "unable to download raw iamge", err)
	}

	finfo := &pb.Download{
		Event: &pb.Download_FileInfo{
			FileInfo: &pb.FileInfo{
				Id:       image.Id,
				FileName: image.Name,
				Owner:    image.Owner,
				Access:   int32(image.Access),
			},
		},
	}

	// Send back file info first.
	if err := stream.Send(finfo); err != nil {
		log.Fatalf("%v.Send(%v) = %v", stream, image.Name, err)
	}

	for cByte := 0; cByte < len(image.Raw); cByte += _ChunkSize {
		var chunk []byte
		if cByte+_ChunkSize > len(image.Raw) {
			chunk = image.Raw[cByte:]
		} else {
			chunk = image.Raw[cByte : cByte+_ChunkSize]
		}

		dchunk := &pb.Download{
			Event: &pb.Download_Chunk{
				Chunk: chunk,
			},
		}

		// Send the file in chunks.
		if err := stream.Send(dchunk); err != nil {
			log.Fatalf("%v.Send(%v) = %v", stream, image.Name, err)
		}
	}

	return nil
}

// ListImages lists the images viewable by the requester.
func (s *repoServer) ListImages(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	// Verify that the user is logged in using token.
	err := s.ss.IsSession(req.Token)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "unable to authenticate ListImages()", err)
	}

	// Get list of images viewable by requester.
	imgs, err := s.ir.List(int(req.Size), req.Sender, req.LastId)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "unable to list images", err)
	}

	// Sender list of images viewable back.
	finfos := make([]*pb.FileInfo, len(imgs))
	for i, img := range imgs {
		finfos[i] = &pb.FileInfo{
			Id:       img.Id,
			FileName: img.Name,
			Owner:    img.Owner,
			Access:   int32(img.Access),
		}
	}

	return &pb.ListResponse{Files: finfos}, nil
}

func newServer() (*repoServer, error) {
	// Load configurations from .env.
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("unable to load .env file: %v", err)
	}

	// Create a UserService
	us, err := mongo.NewUserService(
		os.Getenv("MONGO_URI"),
		os.Getenv("MONGO_DB"),
		os.Getenv("MONGO_ACCS"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create user service: %v", err)
	}
	log.Printf("new UserService created")

	// Create a SessionService
	cdb, err := strconv.Atoi(os.Getenv("CACHE_DB"))
	if err != nil {
		return nil, err
	}

	ss, err := redis.NewSessionService(
		os.Getenv("CACHE_URL"),
		os.Getenv("CACHE_PORT"),
		os.Getenv("CACHE_PASS"),
		cdb,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create session service: %v", err)
	}
	log.Printf("new SessionService created")

	// Create a ImageStorage
	is, err := digitalocean.NewImageStorage(
		os.Getenv("SPACES_KEY"),
		os.Getenv("SPACES_SECRET"),
		os.Getenv("SPACES_ENDPOINT"),
		os.Getenv("SPACES_REGION"),
		os.Getenv("SPACES_BUCKET"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create image storage: %v", err)
	}
	log.Printf("new ImageStorage created")

	// Create a ImageRegistry
	ir, err := mongo.NewImageRegistry(
		is,
		os.Getenv("MONGO_URI"),
		os.Getenv("MONGO_DB"),
		os.Getenv("MONGO_IMGS"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create image registry: %v", err)
	}
	log.Printf("new ImageRegistry created")

	return &repoServer{us: us, ss: ss, ir: ir}, nil
}

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", *serverAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("listening on: %s\n", *serverAddr)

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	server, err := newServer()
	if err != nil {
		log.Fatal(err)
	}

	pb.RegisterRepoServer(grpcServer, server)
	grpcServer.Serve(lis)
}
