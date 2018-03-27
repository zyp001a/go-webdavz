package webdavz

import (
	"fmt"
	"time"
	"context"
	"net/http"
	"path"
	"path/filepath"
	"os"
	"strings"
	"golang.org/x/net/webdav"
)

// Config is the configuration of a WebDAV instance.
type Config struct {
	*User
	Users map[string]*User
}

// ServeHTTP determines if the request is for this plugin, and if all prerequisites are met.
func (c *Config) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u := c.User

	// Gets the correct user for this request.
	username, _, ok := r.BasicAuth()
	if ok {
		if user, ok := c.Users[username]; ok {
			u = user
		}
	}

	// Checks for user permissions relatively to this PATH.
//	if !u.Allowed(r.URL.Path) {
//		w.WriteHeader(http.StatusForbidden)
//		return
//	}

	if r.Method == "HEAD" {
		w = newResponseWriterNoBody(w)
	}

	// If this request modified the files and the user doesn't have permission
	// to do so, return forbidden.
	if (r.Method == "PUT" || r.Method == "POST" || r.Method == "MKCOL" ||
		r.Method == "DELETE" || r.Method == "COPY" || r.Method == "MOVE") &&
		!u.Modify {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Excerpt from RFC4918, section 9.4:
	//
	// 		GET, when applied to a collection, may return the contents of an
	//		"index.html" resource, a human-readable view of the contents of
	//		the collection, or something else altogether.
	//
	// Get, when applied to collection, will return the same as PROPFIND method.
	if r.Method == "GET" {
		info, err := u.Handler.FileSystem.Stat(context.TODO(), r.URL.Path)
		if err == nil && info.IsDir() {
			r.Method = "PROPFIND"
		}
	}

	// Runs the WebDAV.
	u.Handler.ServeHTTP(w, r)
}


// User contains the settings of each user.
type User struct {
	Scope   string
	Modify  bool
	Handler *webdav.Handler
}

// responseWriterNoBody is a wrapper used to suprress the body of the response
// to a request. Mainly used for HEAD requests.
type responseWriterNoBody struct {
	http.ResponseWriter
}

// newResponseWriterNoBody creates a new responseWriterNoBody.
func newResponseWriterNoBody(w http.ResponseWriter) *responseWriterNoBody {
	return &responseWriterNoBody{w}
}

// Header executes the Header method from the http.ResponseWriter.
func (w responseWriterNoBody) Header() http.Header {
	return w.ResponseWriter.Header()
}

// Write suprresses the body.
func (w responseWriterNoBody) Write(data []byte) (int, error) {
	return 0, nil
}

// WriteHeader writes the header to the http.ResponseWriter.
func (w responseWriterNoBody) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}


//Customize Dir

type Dirx struct{
	Path string
	User string
}

//type Dir string;
func (d Dirx) resolve(name string) string {

	// This implementation is based on Dir.Open's code in the standard net/http package.
	if filepath.Separator != '/' && strings.IndexRune(name, filepath.Separator) >= 0 ||
		strings.Contains(name, "\x00") {
		return ""
	}
	dir := d.Path
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, filepath.FromSlash(slashClean(name)))
}

func (d Dirx) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	fmt.Printf("mkdir %s,%s %s\n", name, d.User, time.Now().Format("2006-01-02 15:04:05"));
	
	if name = d.resolve(name); name == "" {
		return os.ErrNotExist
	}
	return os.Mkdir(name, perm)
}

func (d Dirx) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	if(perm > 0){
		fmt.Printf("write %s,%s %s\n", name, d.User, time.Now().Format("2006-01-02 15:04:05"));
	}
	
	if name = d.resolve(name); name == "" {
		return nil, os.ErrNotExist
	}
	f, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (d Dirx) RemoveAll(ctx context.Context, name string) error {
	fmt.Printf("del %s,%s %s\n", name, d.User, time.Now().Format("2006-01-02 15:04:05"));
	if name = d.resolve(name); name == "" {
		return os.ErrNotExist
	}
	if name == filepath.Clean(d.Path) {
		// Prohibit removing the virtual root directory.
		return os.ErrInvalid
	}
	return os.RemoveAll(name)
}

func (d Dirx) Rename(ctx context.Context, oldName, newName string) error {
	fmt.Printf("mv %s->%s,%s %s\n", oldName, newName, d.User, time.Now().Format("2006-01-02 15:04:05"));
	
	if oldName = d.resolve(oldName); oldName == "" {
		return os.ErrNotExist
	}
	if newName = d.resolve(newName); newName == "" {
		return os.ErrNotExist
	}
	if root := filepath.Clean(d.Path); root == oldName || root == newName {
		// Prohibit renaming from or to the virtual root directory.
		return os.ErrInvalid
	}
	return os.Rename(oldName, newName)
}

func (d Dirx) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	if name = d.resolve(name); name == "" {
		return nil, os.ErrNotExist
	}
	return os.Stat(name)
}

// slashClean is equivalent to but slightly more efficient than
// path.Clean("/" + name).
func slashClean(name string) string {
	if name == "" || name[0] != '/' {
		name = "/" + name
	}
	return path.Clean(name)
}

