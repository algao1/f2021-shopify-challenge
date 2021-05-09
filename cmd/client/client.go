package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/algao1/imgrepo"
	"github.com/algao1/imgrepo/mongo"
	"github.com/algao1/imgrepo/proto"
	"google.golang.org/grpc"
)

var (
	serverAddr = flag.String("server_addr", "localhost:10000", "The server address in the format of host:port")
)

func perm(p imgrepo.Permission) string {
	return [...]string{"Public", "Private"}[p]
}

// filteredSearchOfDirectoryTree Walks down a directory tree looking for
// files that match the pattern: re. If a file is found print it out and
// add it to the files list for later user.
func filteredSearchOfDirectoryTree(re *regexp.Regexp, dir string) ([]string, error) {
	files := []string{}

	// Function variable that can be used to filter files based on the pattern.
	walk := func(fn string, fi os.FileInfo, err error) error {
		if !re.MatchString(fn) {
			return nil
		}

		if !fi.IsDir() {
			files = append(files, fn)
		}
		return nil
	}
	filepath.Walk(dir, walk)

	return files, nil
}

func main() {
	flag.Parse()

	log.Printf("connecting to: %s", *serverAddr)

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, *serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := proto.NewRepoClient(conn)
	irc := proto.NewImageRepoClient(client)

	log.Printf("connected to server")

	var lastId string

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := strings.Split(scanner.Text(), " ")
		cmd := input[0]

		if cmd == "reg" && len(input) == 3 {
			err = irc.Register(input[1], input[2])
			if err != nil {
				fmt.Printf("unable to register account: %v\n\n", err)
				continue
			}

			fmt.Println("account registered!")
		} else if cmd == "login" && len(input) == 3 {
			err = irc.Login(input[1], input[2])
			if err != nil {
				fmt.Printf("unable to login to account: %v\n\n", err)
				continue
			}

			fmt.Println("account logged in!")
		} else if cmd == "up" && len(input) >= 4 {
			var files []string

			val, err := strconv.Atoi(input[1])
			if err != nil {
				fmt.Printf("invalid permission: %v\n\n", err)
				continue
			}
			access := imgrepo.Permission(val)

			re, err := regexp.Compile(input[2])
			if err != nil {
				fmt.Printf("invalid regex: %v\n\n", err)
				continue
			}

			for _, dir := range input[3:] {
				nfiles, err := filteredSearchOfDirectoryTree(re, dir)
				if err != nil {
					fmt.Printf("unable to search dir %s: %v\n", dir, err)
				}

				files = append(files, nfiles...)
			}

			start := time.Now()

			fmt.Printf("found %d file(s)\n", len(files))
			for _, file := range files {
				data, err := os.ReadFile(file)
				if err != nil {
					fmt.Printf("unable to open file %s: %v\n\n", file, err)
					continue
				}

				base := filepath.Base(file)
				err = irc.Upload(&imgrepo.Image{Name: base, Owner: irc.Owner, Access: access, Raw: data})
				if err != nil {
					fmt.Printf("unable to upload file %s: %v\n\n", file, err)
					continue
				}

				fmt.Printf("uploaded file: %s\n", file)
			}

			fmt.Printf("uploaded %d files in %v\n", len(files), time.Since(start))
		} else if cmd == "down" && len(input) == 3 {
			img, err := irc.Download(input[1])
			if err != nil {
				fmt.Printf("unable to download image: %v\n\n", err)
				continue
			}

			if _, err := os.Stat(input[2]); os.IsNotExist(err) {
				fmt.Printf("unable to find path: %v\n\n", err)
				continue
			}

			path := filepath.Join(input[2], img.Name)
			os.WriteFile(path, img.Raw, 0666)
			fmt.Printf("downloaded file: %s\n", img.Name)
		} else if cmd == "ls" {
			if len(input) < 2 || input[1] != "-n" {
				lastId = ""
			}

			imgs, err := irc.List(lastId)
			if err != nil {
				fmt.Printf("unable to list images: %v\n\n", err)
				continue
			}

			if len(imgs) > 0 {
				lastId = imgs[len(imgs)-1].Id
			}

			fmt.Printf("found %d image(s)\n", len(imgs))
			for _, img := range imgs {
				t, _ := mongo.GetTime(img)
				fmt.Println(img.Name, img.Owner, perm(img.Access), t.Local().Format("2006-01-02T15:04:05"), img.Id)
			}
		} else {
			fmt.Printf("invalid command: %s\n", input)
		}

		fmt.Println()
	}
}
