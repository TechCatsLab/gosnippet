package run

import (
	"runtime"
	"os/exec"
	"io/ioutil"
	"fmt"
)

type File struct {
	FileRoot string
	FileName string
}

const (
	goOS string = runtime.GOOS
	WINDOWS = "windows"
	LINUX = "linux"
)

var systemMap = map[string]func(f File) (name string, arg string) {
	WINDOWS : windows,
	LINUX : linux,
}

func windows(f File) (name string, arg string) {
	if f.FileRoot == "" {
		name = f.FileName
	} else {
		name = f.FileRoot + "\\" + f.FileName
	}
	return
}

func linux (f File) (name string, arg string) {
	name = "sh"
	if f.FileRoot == "" {
		arg = f.FileName
	} else {
		arg = f.FileRoot + "/" + f.FileName
	}
	return
}

func (f *File)RunCommand() {
	nameCommand, argCommand := systemMap[goOS](*f)
	cmd := exec.Command(nameCommand, argCommand)
	errPipe, _ := cmd.StderrPipe()
	outPipe, _ := cmd.StdoutPipe()
	cmd.Start()
	rErrPipe, _ := ioutil.ReadAll(errPipe)
	rOutPipe, _ := ioutil.ReadAll(outPipe)
	cmd.Wait()
	fmt.Println(string(rErrPipe), "\n", string(rOutPipe))
}