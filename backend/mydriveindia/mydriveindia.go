package mydriveindia

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/rclone/rclone/backend/mydriveindia/api"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/rclone/rclone/fs/fshttp"
	"github.com/rclone/rclone/fs/hash"
	"github.com/rclone/rclone/fs/list"
	"github.com/rclone/rclone/lib/rest"
)

func init() {
	fs.Register(&fs.RegInfo{
		Name:        "mydriveindia",
		Description: "MyDriveIndia",
		NewFs:       NewFs,
		Options: []fs.Option{{
			Name: "session_id",
			Help: `Session ID for MyPayIndia.

You can get it from the browser's developer tools after logging in to MyPayIndia/MyDriveIndia.`,
			Sensitive: true,
		}},
	})
}

type Options struct {
	SessionID string `config:"session_id"`
}

type Fs struct {
	name string
	root string
	srv  *rest.Client
}

type Object struct {
	fs      *Fs       // what this object is part of
	remote  string    // the remote path
	size    int64     // size of the object
	modTime time.Time // modification time of the object
}

func NewFs(ctx context.Context, name, root string, m configmap.Mapper) (fs.Fs, error) {
	// Parse config into Options struct
	opt := new(Options)
	err := configstruct.Set(m, opt)
	if err != nil {
		return nil, err
	}

	client := fshttp.NewClient(ctx)

	if root != "" && !strings.HasSuffix(root, "/") {
		root += "/"
	}

	f := &Fs{
		name: name,
		root: root,
		srv:  rest.NewClient(client).SetRoot("https://drive.mypayindia.com/api/").SetHeader("Authorization", fmt.Sprintf("Bearer %s", opt.SessionID)),
	}
	return f, nil
}

// Name of the remote (as passed into NewFs)
func (f *Fs) Name() string {
	return f.name
}

// Root of the remote (as passed into NewFs)
func (f *Fs) Root() string {
	return f.root
}

// String converts this Fs into a string
func (f *Fs) String() string {
	return fmt.Sprintf("mydriveindia root '%s'", f.root)
}

// Features returns the optional features of this Fs
func (f *Fs) Features() *fs.Features {
	return &fs.Features{
		CaseInsensitive:         true,
		CanHaveEmptyDirectories: true,
	}
}

func (f *Fs) Precision() time.Duration {
	return fs.ModTimeNotSupported
}

func (f *Fs) Hashes() hash.Set {
	return hash.Set(hash.None)
}

// List the objects and directories in dir into entries.  The
// entries can be returned in any order but should be for a
// complete directory.
//
// dir should be "" to list the root, and should not have
// trailing slashes.
//
// This should return ErrDirNotFound if the directory isn't
// found.
func (f *Fs) List(ctx context.Context, dir string) (entries fs.DirEntries, err error) {
	return list.WithListP(ctx, dir, f)
}

// ListP lists the objects and directories of the Fs starting
// from dir non recursively into out.
//
// dir should be "" to start from the root, and should not
// have trailing slashes.
//
// This should return ErrDirNotFound if the directory isn't
// found.
//
// It should call callback for each tranche of entries read.
// These need not be returned in any particular order.  If
// callback returns an error then the listing will stop
// immediately.
func (f *Fs) ListP(ctx context.Context, dir string, callback fs.ListRCallback) error {
	// add slash suffix for file search if dir isn't root
	if dir != "" && !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	// change root
	dir = f.root + dir
	fs.Debugf("ListP", "resolved dir=%q", dir)

	list := list.NewHelper(callback)
	_, err := f.listAll(ctx, func(item *api.FileInfo) bool {
		if strings.HasSuffix(item.Path, "/.directory") && !strings.ContainsAny(strings.TrimSuffix(strings.TrimPrefix(item.Path, dir), "/.directory"), "/") {
			remote := strings.TrimPrefix(item.Path, f.root)
			remote = strings.TrimSuffix(remote, "/.directory")
			fs.Debugf("ListP", "remote=%q", remote)
			// ignore this directory's dir marker
			if remote+"/" == dir {
				return false
			}
			if remote == ".directory" {
				return false
			}
			d := fs.NewDir(remote, item.CreatedAt)
			list.Add(d)
		}
		if strings.HasPrefix(item.Path, dir) && !strings.ContainsAny(strings.TrimPrefix(item.Path, dir), "/") {
			list.Add(f.itemToObject(item))
		}
		return false
	})
	if err != nil {
		return err
	}
	return list.Flush()
}

func (f *Fs) Mkdir(ctx context.Context, dir string) error {
	fs.Debug(dir, "Mkdir hit: TODO")
	return fs.ErrorNotImplemented
}

// NewObject finds the Object at remote.  If it can't be found it returns the error fs.ErrorObjectNotFound.
func (f *Fs) NewObject(ctx context.Context, remote string) (fs.Object, error) {
	fs.Debug(remote, "NewObject hit: TODO")
	return nil, fs.ErrorNotImplemented
}

// Put the object
func (f *Fs) Put(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	// TODO: implement Put, make sure it deletes the object if it exists because MDI doesn't support in-place file updates
	fs.Debug(src, "Put hit: TODO")
	return nil, fs.ErrorNotImplemented
}

func (f *Fs) Rmdir(ctx context.Context, dir string) error {
	fs.Debug(dir, "Rmdir hit: TODO")
	return fs.ErrorNotImplemented
}

type listAllFn func(*api.FileInfo) bool

// Returns all objects in the filesystem from the API.
func (f *Fs) listAll(ctx context.Context, fn listAllFn) (found bool, err error) {
	opts := rest.Opts{
		Method:     "GET",
		Path:       "files",
		Parameters: url.Values{},
	}

	var result api.FilesResponse
	_, err = f.srv.CallJSON(ctx, &opts, nil, &result)
	if err != nil {
		return found, err
	}

	for _, item := range result.Files {
		if fn(&item) {
			found = true
			break
		}
	}

	return found, err
}

func (f *Fs) itemToObject(item *api.FileInfo) fs.Object {
	remote := strings.TrimPrefix(item.Path, f.root)
	fs.Debugf("itemToObject", "remote=%q", remote)
	o := &Object{
		fs:      f,
		remote:  remote,
		size:    item.Size,
		modTime: item.CreatedAt,
	}
	return o
}

// Fs returns read only access to the Fs that this object is part of
func (o *Object) Fs() fs.Info {
	return o.fs
}

// String returns a description of the Object
func (o *Object) String() string {
	if o == nil {
		return "<nil>"
	}
	return o.remote
}

// Remote returns the remote path
func (o *Object) Remote() string {
	return o.remote
}

// ModTime returns the modification date of the file
// It should return a best guess if one isn't available
func (o *Object) ModTime(context.Context) time.Time {
	return o.modTime
}

// Size returns the size of the file
func (o *Object) Size() int64 {
	return o.size
}

// Hash returns the selected checksum of the file
// If no checksum is available it returns ""
func (o *Object) Hash(ctx context.Context, ty hash.Type) (string, error) {
	return "", nil
}

// Storable says whether this object can be stored
func (o *Object) Storable() bool {
	return true
}

// SetModTime sets the metadata on the object to set the modification date
func (o *Object) SetModTime(ctx context.Context, t time.Time) error {
	return fs.ErrorNotImplemented
}

// Open opens the file for read.  Call Close() on the returned io.ReadCloser
func (o *Object) Open(ctx context.Context, options ...fs.OpenOption) (io.ReadCloser, error) {
	return nil, fs.ErrorNotImplemented
}

// Update in to the object with the modTime given of the given size
//
// When called from outside an Fs by rclone, src.Size() will always be >= 0.
// But for unknown-sized objects (indicated by src.Size() == -1), Upload should either
// return an error or update the object properly (rather than e.g. calling panic).
func (o *Object) Update(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) error {
	return fs.ErrorNotImplemented
}

// Removes this object
func (o *Object) Remove(ctx context.Context) error {
	return fs.ErrorNotImplemented
}

// Check the interfaces are satisfied
var (
	_ fs.Fs     = &Fs{}
	_ fs.Object = &Object{}
)
