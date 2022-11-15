package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/studio-b12/gowebdav"
)

// function to check if file exists
func doesFileExist(fileName string) bool {
	_, error := os.Stat(fileName)

	// check if error is "file not exists"
	if os.IsNotExist(error) {
		fmt.Printf("%v file does not exist\n", fileName)
		return false
	}
	return true
}

func get_local_path(base_dir string) string {
	path := "~" + base_dir

	if strings.HasPrefix(path, "~/") {
		dirname, _ := os.UserHomeDir()
		path = filepath.Join(dirname, path[2:])
	}

	return path
}

func download_file(c *gowebdav.Client, base_dir string, file_name string, server_time time.Time) {
	reader, _ := c.ReadStream(base_dir + file_name)

	full_path := filepath.Join(get_local_path(base_dir), file_name)
	fmt.Println("Downloading new File ", full_path)
	f, _ := os.Create(full_path)
	defer f.Close()

	io.Copy(f, reader)

	err := os.Chtimes(full_path, server_time, server_time)
	if err != nil {
		fmt.Println(err)
	}
}

func upload_file(c *gowebdav.Client, base_dir string, file_name string) {
	full_path := filepath.Join(get_local_path(base_dir), file_name)

	fmt.Println("Uploading new File ", full_path)
	bytes, _ := ioutil.ReadFile(full_path)

	c.Write(filepath.Join(base_dir, file_name), bytes, 0644)

	fil, _ := os.Open(full_path)
	defer fil.Close()

	c.WriteStream(filepath.Join(base_dir, file_name), fil, 0644)
}


func upload_a_local_folder(c *gowebdav.Client, base_dir string, local_folder_name string, local_base_path string) {
	fmt.Println("uploading local folder ", base_dir, local_folder_name)

	err := c.Mkdir(base_dir+local_folder_name, 0644)
	if err != nil {
		panic("error creating remote folder")
	}

	fils, err := os.ReadDir(filepath.Join(local_base_path, local_folder_name))
	if err != nil {
		panic("error reading local folder" + local_base_path + local_folder_name)
	}

	for _, file := range fils {
		if file.IsDir() {

			fmt.Println("-- ", base_dir+local_folder_name)
			check_folder(c, base_dir+local_folder_name+"/")

		} else {
			upload_file(c, base_dir+local_folder_name+"/", file.Name())
		}
	}
}

func check_folder(c *gowebdav.Client, base_dir string) {
	println("checking base dir = ", base_dir)

	sync := make(map[string]time.Time)

	files, e := c.ReadDir(base_dir)
	if e != nil {
		fmt.Println(e)
	}

	path := get_local_path(base_dir)

	for _, file := range files {
		if file.IsDir() {
			check_folder(c, base_dir+file.Name()+"/")
		} else {
			full_path := filepath.Join(path, file.Name())
			if doesFileExist(full_path) != true {
				download_file(c, base_dir, file.Name(), file.ModTime())
			}
			sync[base_dir+file.Name()] = file.ModTime()
			// fmt.Println(base_dir + file.Name(), file.Size())
			// if deu, ok := file.(gowebdav.File); ok {
			// 	// fmt.Println(base_dir+file.Name(), deu.Size(), deu.ETag())
			// }
		}
	}

	fils, err := os.ReadDir(path)
	if err != nil {
		fmt.Println("Creating a folder", path)
		os.Mkdir(path, os.ModePerm)
		for _, file := range files {
			download_file(c, base_dir, file.Name(), file.ModTime())
		}
	}

	for _, file := range fils {
		if file.IsDir() {

			_, err := c.Stat(filepath.Join(base_dir, file.Name()))
			if err != nil {
				fmt.Println("cretate a remote dir")
				upload_a_local_folder(c, base_dir, file.Name(), path)

			}

		} else {
			full_path := filepath.Join(path, file.Name())
			fi, _ := os.Stat(full_path)
			md := fi.ModTime()
			k := sync[base_dir+file.Name()]
			if k.After(md) {
				download_file(c, base_dir, file.Name(), k)
			}
			if k.Before(md) {
				upload_file(c, base_dir, file.Name())
			}
		}
	}
}

func main() {
	server := os.Getenv("WEBDAV_SERVER")
	user := os.Getenv("WEBDAV_USER")
	password := os.Getenv("WEBDAV_PASSWORD")

	if server == "" {
		panic("No server address set")
	}

	if user == "" {
		panic("No server address set")
	}

	if password == "" {
		panic("No server address set")
	}
	c := gowebdav.NewClient(server, user, password)

	check_folder(c, "/notes/")
}
