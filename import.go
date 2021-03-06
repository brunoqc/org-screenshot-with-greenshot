package main

import (
	"flag"
	"fmt"
	"github.com/lxn/walk"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

var (
	send = flag.String("send", "", "path of the capture to send")
	port = flag.Int("port", 64081, "TCP port to use")
)

type Path struct {
	OrgFilePath string
	ChanQuit    chan int
}

func (p *Path) Send(path string, reply *int) error {
	defer close(p.ChanQuit)

	_, errCopyFile := CopyFile(path, p.OrgFilePath)
	if errCopyFile != nil {
		return errCopyFile
	}

	return os.Remove(path)
}

func CopyFile(src, dst string) (int64, error) {
	sf, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer sf.Close()
	df, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer df.Close()
	return io.Copy(df, sf)
}

func main() {
	chanQuit := make(chan int)

	flag.Parse()

	switch {
	case *send != "": // client (called by Greenshot)
		client, err := rpc.DialHTTP("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
		if err != nil {
			walk.MsgBox(nil, "Error", "Can't connect to server", walk.MsgBoxIconExclamation)
			return
		}

		err = client.Call("Path.Send", *send, nil)
		if err != nil {
			walk.MsgBox(nil, "Error", "Error sending the path", walk.MsgBoxIconExclamation)
			return
		}
	case len(os.Args) != 2:
		fmt.Println("Must be called with one argument")
	default: // server (called by org-screenshot)
		path := &Path{
			OrgFilePath: os.Args[1],
			ChanQuit:    chanQuit,
		}

		rpc.Register(path)
		rpc.HandleHTTP()
		l, e := net.Listen("tcp", fmt.Sprintf(":%d", *port))
		if e != nil {
			walk.MsgBox(nil, "Error", fmt.Sprintf("Can't listen on port %d (TCP)", *port), walk.MsgBoxIconExclamation)
			return
		}
		go http.Serve(l, nil)
		<-chanQuit
	}
}
