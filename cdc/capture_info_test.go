package cdc

import (
	"context"
	"net/url"
	"time"

	"github.com/pingcap/check"
	"github.com/pingcap/ticdc/cdc/kv"
	"github.com/pingcap/ticdc/cdc/model"
	"github.com/pingcap/ticdc/pkg/etcd"
	"github.com/pingcap/ticdc/pkg/util"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/embed"
	"golang.org/x/sync/errgroup"
)

type captureInfoSuite struct {
	etcd      *embed.Etcd
	clientURL *url.URL
	client    kv.CDCEtcdClient
	ctx       context.Context
	cancel    context.CancelFunc
	errg      *errgroup.Group
}

var _ = check.Suite(&captureInfoSuite{})

func (ci *captureInfoSuite) SetUpTest(c *check.C) {
	dir := c.MkDir()
	var err error
	ci.clientURL, ci.etcd, err = etcd.SetupEmbedEtcd(dir)
	c.Assert(err, check.IsNil)
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{ci.clientURL.String()},
		DialTimeout: 3 * time.Second,
	})
	c.Assert(err, check.IsNil)
	ci.client = kv.NewCDCEtcdClient(client)
	ci.ctx, ci.cancel = context.WithCancel(context.Background())
	ci.errg = util.HandleErrWithErrGroup(ci.ctx, ci.etcd.Err(), func(e error) { c.Log(e) })
}

func (ci *captureInfoSuite) TearDownTest(c *check.C) {
	ci.etcd.Close()
	ci.cancel()
	err := ci.errg.Wait()
	if err != nil {
		c.Errorf("Error group error: %s", err)
	}
}

func (ci *captureInfoSuite) TestPutDeleteGet(c *check.C) {
	ctx := context.Background()

	id := "1"

	// get a not exist capture
	info, err := ci.client.GetCaptureInfo(ctx, id)
	c.Assert(err, check.Equals, model.ErrCaptureNotExist)
	c.Assert(info, check.IsNil)

	// create
	info = &model.CaptureInfo{
		ID: id,
	}
	err = ci.client.PutCaptureInfo(ctx, info)
	c.Assert(err, check.IsNil)

	// get again,
	getInfo, err := ci.client.GetCaptureInfo(ctx, id)
	c.Assert(err, check.IsNil)
	c.Assert(getInfo, check.DeepEquals, info)

	// delete it
	err = ci.client.DeleteCaptureInfo(ctx, id)
	c.Assert(err, check.IsNil)
	// get again should not exist
	info, err = ci.client.GetCaptureInfo(ctx, id)
	c.Assert(err, check.Equals, model.ErrCaptureNotExist)
	c.Assert(info, check.IsNil)
}
