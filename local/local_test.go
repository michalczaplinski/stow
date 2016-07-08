package local_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cheekybits/is"
	"github.com/graymeta/stow"
	"github.com/graymeta/stow/local"
	"github.com/graymeta/stow/test"
)

func setup() (string, func() error, error) {
	done := func() error { return nil } // noop
	dir, err := ioutil.TempDir("testdata", "stow")
	if err != nil {
		return dir, done, err
	}
	done = func() error {
		return os.RemoveAll(dir)
	}
	// add some "containers"
	err = os.Mkdir(filepath.Join(dir, "one"), 0777)
	if err != nil {
		return dir, done, err
	}
	err = os.Mkdir(filepath.Join(dir, "two"), 0777)
	if err != nil {
		return dir, done, err
	}
	err = os.Mkdir(filepath.Join(dir, "three"), 0777)
	if err != nil {
		return dir, done, err
	}

	// add three items
	err = ioutil.WriteFile(filepath.Join(dir, "three", "item1"), []byte("3.1"), 0777)
	if err != nil {
		return dir, done, err
	}
	err = ioutil.WriteFile(filepath.Join(dir, "three", "item2"), []byte("3.2"), 0777)
	if err != nil {
		return dir, done, err
	}
	err = ioutil.WriteFile(filepath.Join(dir, "three", "item3"), []byte("3.3"), 0777)
	if err != nil {
		return dir, done, err
	}

	// make testpath absolute
	absdir, err := filepath.Abs(dir)
	if err != nil {
		return dir, done, err
	}
	return absdir, done, nil
}

func TestStow(t *testing.T) {
	is := is.New(t)

	dir, err := ioutil.TempDir("testdata", "stow")
	is.NoErr(err)
	defer os.RemoveAll(dir)
	cfg := stow.ConfigMap{"path": dir}

	test.All(t, "local", cfg)
}

func TestContainers(t *testing.T) {
	is := is.New(t)
	testDir, teardown, err := setup()
	is.NoErr(err)
	defer teardown()

	cfg := stow.ConfigMap{"path": testDir}

	l, err := stow.Dial(local.Kind, cfg)
	is.NoErr(err)
	is.OK(l)

	items, cursor, err := l.Containers("", stow.CursorStart)
	is.NoErr(err)
	is.Equal(cursor, "")
	is.OK(items)

	is.Equal(len(items), 3)
	isDir(is, items[0].ID())
	is.Equal(items[0].Name(), "one")
	isDir(is, items[1].ID())
	is.Equal(items[1].Name(), "three")
	isDir(is, items[2].ID())
	is.Equal(items[2].Name(), "two")
}

func TestContainersPrefix(t *testing.T) {
	is := is.New(t)
	testDir, teardown, err := setup()
	is.NoErr(err)
	defer teardown()

	cfg := stow.ConfigMap{"path": testDir}

	l, err := stow.Dial(local.Kind, cfg)
	is.NoErr(err)
	is.OK(l)

	containers, cursor, err := l.Containers("t", stow.CursorStart)
	is.NoErr(err)
	is.OK(containers)
	is.Equal(cursor, "")

	is.Equal(len(containers), 2)
	isDir(is, containers[0].ID())
	is.Equal(containers[0].Name(), "three")
	isDir(is, containers[1].ID())
	is.Equal(containers[1].Name(), "two")

	cthree, err := l.Container(containers[0].ID())
	is.NoErr(err)
	is.Equal(cthree.Name(), "three")
}

func TestContainer(t *testing.T) {
	is := is.New(t)
	testDir, teardown, err := setup()
	is.NoErr(err)
	defer teardown()

	cfg := stow.ConfigMap{"path": testDir}

	l, err := stow.Dial(local.Kind, cfg)
	is.NoErr(err)
	is.OK(l)

	containers, cursor, err := l.Containers("t", stow.CursorStart)
	is.NoErr(err)
	is.OK(containers)
	is.Equal(cursor, "")

	is.Equal(len(containers), 2)
	isDir(is, containers[0].ID())

	cthree, err := l.Container(containers[0].ID())
	is.NoErr(err)
	is.Equal(cthree.Name(), "three")

}

func TestCreateContainer(t *testing.T) {
	is := is.New(t)
	testDir, teardown, err := setup()
	is.NoErr(err)
	defer teardown()

	cfg := stow.ConfigMap{"path": testDir}

	l, err := stow.Dial(local.Kind, cfg)
	is.NoErr(err)
	is.OK(l)

	c, err := l.CreateContainer("new_test_container")
	is.NoErr(err)
	is.OK(c)
	is.Equal(c.ID(), filepath.Join(testDir, "new_test_container"))
	is.Equal(c.Name(), "new_test_container")

	containers, cursor, err := l.Containers("new", stow.CursorStart)
	is.NoErr(err)
	is.OK(containers)
	is.Equal(cursor, "")

	is.Equal(len(containers), 1)
	isDir(is, containers[0].ID())
	is.Equal(containers[0].Name(), "new_test_container")
}

func TestCreateItem(t *testing.T) {
	is := is.New(t)
	testDir, teardown, err := setup()
	is.NoErr(err)
	defer teardown()

	cfg := stow.ConfigMap{"path": testDir}
	l, err := stow.Dial(local.Kind, cfg)
	is.NoErr(err)
	is.OK(l)

	containers, cursor, err := l.Containers("t", stow.CursorStart)
	is.NoErr(err)
	is.OK(containers)
	c1 := containers[0]
	items, cursor, err := c1.Items(stow.CursorStart)
	is.NoErr(err)
	is.Equal(cursor, "")
	beforecount := len(items)

	content := "new item contents"
	newitem, err := c1.Put("new_item", strings.NewReader(content), int64(len(content)))
	is.NoErr(err)
	is.OK(newitem)
	is.Equal(newitem.Name(), "new_item")

	// get the container again
	containers, cursor, err = l.Containers("t", stow.CursorStart)
	is.NoErr(err)
	is.OK(containers)
	is.Equal(cursor, "")
	c1 = containers[0]
	items, cursor, err = c1.Items(stow.CursorStart)
	is.NoErr(err)
	is.Equal(cursor, "")
	aftercount := len(items)

	is.Equal(aftercount, beforecount+1)

	// get new item
	item := items[len(items)-1]
	md5, err := item.MD5()
	is.NoErr(err)
	is.Equal(md5, "1d4b28e33c8bfcfdb75e116ed2319632")
	etag, err := item.ETag()
	is.NoErr(err)
	is.OK(etag)
	r, err := item.Open()
	is.NoErr(err)
	defer r.Close()
	itemContents, err := ioutil.ReadAll(r)
	is.NoErr(err)
	is.Equal("new item contents", string(itemContents))

}

func TestItems(t *testing.T) {
	is := is.New(t)
	testDir, teardown, err := setup()
	is.NoErr(err)
	defer teardown()

	cfg := stow.ConfigMap{"path": testDir}

	l, err := stow.Dial(local.Kind, cfg)
	is.NoErr(err)
	is.OK(l)

	containers, cursor, err := l.Containers("t", stow.CursorStart)
	is.NoErr(err)
	is.OK(containers)
	is.Equal(cursor, "")
	three, err := l.Container(containers[0].ID())
	is.NoErr(err)
	items, cursor, err := three.Items(stow.CursorStart)
	is.NoErr(err)
	is.OK(items)
	is.Equal(cursor, "")

	is.Equal(len(items), 3)
	is.Equal(items[0].ID(), filepath.Join(containers[0].ID(), "item1"))
	is.Equal(items[0].Name(), "item1")
}

func TestByURL(t *testing.T) {
	is := is.New(t)
	testDir, teardown, err := setup()
	is.NoErr(err)
	defer teardown()

	cfg := stow.ConfigMap{"path": testDir}

	l, err := stow.Dial(local.Kind, cfg)
	is.NoErr(err)
	is.OK(l)

	containers, cursor, err := l.Containers("t", stow.CursorStart)
	is.NoErr(err)
	is.OK(containers)
	is.Equal(cursor, "")

	three, err := l.Container(containers[0].ID())
	is.NoErr(err)
	items, cursor, err := three.Items(stow.CursorStart)
	is.NoErr(err)
	is.OK(items)
	is.Equal(cursor, "")
	is.Equal(len(items), 3)

	item1 := items[0]

	// make sure we know the kind by URL
	kind, err := stow.KindByURL(item1.URL())
	is.NoErr(err)
	is.Equal(kind, local.Kind)

	i, err := l.ItemByURL(item1.URL())
	is.NoErr(err)
	is.OK(i)
	is.Equal(i.ID(), item1.ID())
	is.Equal(i.Name(), item1.Name())
	is.Equal(i.URL().String(), item1.URL().String())

}
func TestItemReader(t *testing.T) {
	is := is.New(t)
	testDir, teardown, err := setup()
	is.NoErr(err)
	defer teardown()

	cfg := stow.ConfigMap{"path": testDir}
	l, err := stow.Dial(local.Kind, cfg)
	is.NoErr(err)
	is.OK(l)
	containers, cursor, err := l.Containers("t", stow.CursorStart)
	is.NoErr(err)
	is.OK(containers)
	is.Equal(cursor, "")
	three, err := l.Container(containers[0].ID())

	items, cursor, err := three.Items(stow.CursorStart)
	is.NoErr(err)
	is.Equal(cursor, "")
	item1 := items[0]

	rc, err := item1.Open()
	defer rc.Close()
	is.NoErr(err)
	b, err := ioutil.ReadAll(rc)
	is.NoErr(err)
	is.Equal("3.1", string(b))

}

func isDir(is is.I, path string) {
	info, err := os.Stat(path)
	is.NoErr(err)
	is.True(info.IsDir())
}
